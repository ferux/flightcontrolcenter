/*
Package keeper communicates with any device via TCP and allows
to look after it, allowing duplex communication between it.

Here will be list of must-have functions that should be implemented:
...

Here will be list of optional functions that may be implemented:
...


*/
package keeper

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/keeper/talk"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/pubsub"

	"go.uber.org/zap"
)

const KeeperTopic = "keeper_topic"

type serverState uint8

const (
	serverStateRunning serverState = iota + 1
	serverStateShutdown
)

type Server struct {
	l      net.Listener
	r      Repo
	logger *zap.Logger

	serverState serverState
	conns       map[model.DeviceID]*Conn
	h           map[talk.MessageType]handler
	subs        *pubsub.Core

	mu sync.RWMutex
}

func New(cfg config.Keeper, r Repo, logger *zap.Logger, subs *pubsub.Core) (*Server, error) {
	tlsCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("loading x509 keypair: %w", err)
	}

	var l net.Listener
	l, err = tls.Listen("tcp4", cfg.Listen, &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ServerName:   cfg.Name,
	})

	if err != nil {
		return nil, fmt.Errorf("opening tls listener: %w", err)
	}

	s := &Server{
		l:      l,
		r:      r,
		logger: logger.With(zap.String("pkg", "keeper")),
		subs:   subs,

		serverState: serverStateRunning,
		conns:       make(map[model.DeviceID]*Conn),
	}
	go s.loop()

	return s, nil
}

const defaultTimeout = time.Second * 30

func (s *Server) loop() {
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		if err := s.r.MarkAll(ctx, model.DeviceStateOffline); err != nil {
			s.logger.Error("marking all devices offline", zap.Error(err))
		}
	}()

	for {
		conn, err := s.l.Accept()
		if err != nil {
			s.logger.Error("accepting connection", zap.Error(err))
			return
		}
		dconn, err := HandshakeConnection(defaultTimeout, conn, s.r)
		if err != nil {
			var perr PermamentError
			soft := !errors.As(err, &perr)
			errDeny := dconn.DenyConnection(context.Background(), err.Error(), soft)
			if errDeny != nil {
				s.logger.Warn("error denying connection", zap.Error(err))
			}
			continue
		}

		s.addConnection(dconn)
	}
}

func (s *Server) getConnection(deviceID model.DeviceID) *Conn {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conn, ok := s.conns[deviceID]
	if !ok {
		return nil
	}

	return conn
}

func (s *Server) addConnection(conn *Conn) {
	logger := s.logger.With(zap.Uint64("device_id", uint64(conn.device.ID)))

	dconn := s.getConnection(conn.device.ID)
	if dconn != nil {
		logger.Warn("connection exists, closing it")
		errClose := dconn.Close()
		if errClose != nil {
			logger.Error("closing connection", zap.Error(errClose))
		}
	}

	s.subs.Notify(KeeperTopic, conn.device.ID, model.DeviceStateOnline)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[conn.device.ID] = conn
}

func (s *Server) removeConnection(conn *Conn) {
	logger := s.logger.With(zap.Uint64("device_id", uint64(conn.device.ID)))

	dconn := s.getConnection(conn.device.ID)
	if dconn == nil {
		return
	}

	err := conn.Close()
	if err != nil {
		logger.Error("closing connection")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	errMark := s.r.Mark(ctx, conn.device.ID, model.DeviceStateOffline)
	if errMark != nil {
		logger.Error("marking device offline", zap.Error(errMark))
	}

	s.subs.Notify(KeeperTopic, conn.device.ID, model.DeviceStateOffline)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conns, conn.device.ID)
}

// HandleConnection handshakes connection, applies entity info to it and performs
// remove actions in case connection gone.
func (s *Server) HandleConnection(conn *Conn) {
	defer func() {
		// if server is shuting down, we don't need to perform any
		// extra action because Shutdown(...) will do everything for us.
		if s.serverState == serverStateShutdown {
			return
		}

		s.removeConnection(conn)
	}()

	logger := s.logger.With(zap.Uint64("device_id", uint64(conn.device.ID)))

	var (
		p   Packet
		err error
	)

	ctx := fcontext.WithZap(context.Background(), logger)
	ctx = fcontext.WithDeviceID(ctx, conn.device.ID)

	for {
		p, err = conn.Read()
		if err != nil {
			logger.Error("reading packet", zap.Error(err))
			return
		}

		pctx := fcontext.WithDeviceRequestID(ctx, p.Header.RequestID)
		logger.Debug(
			"incoming packet",
			zap.Uint64("req_id", p.Header.RequestID),
			zap.String("msg_type", p.Header.MessageType.String()),
			zap.Uint64("size", p.Header.BodySize),
		)

		handler, ok := s.h[p.Header.MessageType]
		if !ok {
			logger.Warn("handler not found", zap.String("handler", p.Header.MessageType.String()))
			continue
		}

		err = handler(pctx, p.Body, conn)
		if err != nil {
			logger.Warn("handler error", zap.Uint64("request_id", p.Header.RequestID), zap.Error(err))
		}
	}
}

// Shutdown performs graceful shutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.serverState = serverStateShutdown
	s.mu.RUnlock()

	done := make(chan struct{})

	go func(ctx context.Context, done chan struct{}) {
		s.shutdown(ctx)
		close(done)
	}(ctx, done)

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) shutdown(ctx context.Context) {
	err := s.l.Close()
	if err != nil {
		s.logger.Error("closing listener", zap.Error(err))
	}

	err = s.r.MarkAll(ctx, model.DeviceStateOffline)
	if err != nil {
		s.logger.Error("marking devices offline", zap.Error(err))
	}
}
