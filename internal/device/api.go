package device

import (
	"context"
	"encoding/gob"
	"net"
	"sync"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/ferux/flightcontrolcenter/internal/model"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

const defaultCloseLimit = 10

type API struct {
	addr   string
	l      net.Listener
	h      map[MessageType]handler
	conns  map[model.DeviceID]*Connection
	connMu sync.RWMutex

	log        logger.Logger
	bufferPool sync.Pool
}

func (a *API) getBuffer() []byte {
	b, _ := a.bufferPool.Get().([]byte)
	if b == nil {
		b = make([]byte, 0, 1024)
	}

	return b
}

func (a *API) returnBuffer(b []byte) {
	b = b[:]
	a.bufferPool.Put(b)
}

func (a *API) Run() error {
	a.log.Debug("starting service")
	l, err := net.Listen("tcp", a.addr)
	if err != nil {
		return errors.Wrap(err, "unable to listen address")
	}

	a.l = l
	for {
		conn, err := l.Accept()
		if err == nil {
			a.log.Debugf("unable to accept connection: %v", err)
			break
		}

		go a.handleConnection(&Connection{conn: conn})
	}

	return nil
}

func (a *API) newContext() context.Context {
	ctx := fcontext.WithLogger(context.Background(), a.log)
	return fcontext.WithRequestID(ctx, uuid.New())
}

func (a *API) handleConnection(c *Connection) {
	a.log.Debugf("connection from %s accepted", c.RemoteAddr().String())

	ctx := a.newContext()

	go func() {
		b := a.getBuffer()
		_, err := c.Read(b)
		if err != nil {
			a.log.Debugf("unable to read from %s: %v", c.RemoteAddr().String(), err)
			return
		}
		var msg Message
		err = gob.NewDecoder(c.conn).Decode(&msg)
		if err != nil {
			a.log.Debugf("unable to decode message: %v", err)
		}

		if c.device == nil {
			ctx = fcontext.WithDeviceID(ctx, c.device.ID)
		}

		err = a.h[msg.Type](ctx, msg.Data)
		if err != nil {
			a.log.Debugf("unable to handle message: %v", err)
			_ = c.Close()
			if c.device != nil {
				delete(a.conns, c.device.ID)
			}
		}
	}()
	var hr = HelloRequest{
		ServerVersion: "0.0.1",
	}
	err := c.sendHelloRequest(ctx, hr)
	if err != nil {
		a.log.Errf("unable to send hello request: %v", err)
	}

	// TODO: register device
}

// Shutdown closes all active connections.
func (a *API) Shutdown(ctx context.Context) error {
	var limiter = make(chan struct{}, defaultCloseLimit)
	for _, v := range a.conns {
		select {
		case limiter <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		limiter <- struct{}{}
		go func(c *Connection) {
			errClose := c.Close()
			if errClose != nil {
				a.log.Debugf("unable to close connection for device %d: %v", c.device.ID, errClose)
			}
			<-limiter
		}(v)
	}

	close(limiter)

	errClose := a.l.Close()
	if errClose != nil {
		return errors.Wrap(errClose, "unable to close listener")
	}

	return nil
}

type Message struct {
	Type MessageType
	Data []byte
}

// MessageType determines how to handle the message
type MessageType uint64

type handler func(ctx context.Context, message []byte) error
