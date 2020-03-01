package fccgob

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/google/uuid"

	"github.com/rs/zerolog"
)

// MessageKind describes type of message encoded into body.
type MessageKind uint8

const (
	MessageKindUnknown = iota
	MessageKindLog
	MessageKindNotify
	MessageKindOK
	MessageKindFail
)

func (k MessageKind) String() string {
	switch k {
	case MessageKindUnknown:
		return "kind_unknown"
	case MessageKindLog:
		return "kind_log"
	case MessageKindNotify:
		return "kind_notify"
	case MessageKindOK:
		return "kind_ok"
	case MessageKindFail:
		return "kind_fail"
	default:
		return "undefind"
	}
}

// Message is transffered over the wire packet.
type Message struct {
	RequestID string
	Kind      MessageKind
	Data      []byte
}

// Serve starts to listen incoming connections.
func Serve(ctx context.Context, cfg config.GOB, logger zerolog.Logger, h map[MessageKind]Handler) (err error) {
	gob.Register(Message{})

	l, err := net.Listen("tcp4", cfg.Listen)
	if err != nil {
		return fmt.Errorf("listening: %w", err)
	}

	go func() {
		<-ctx.Done()

		errClose := l.Close()
		if errClose != nil {
			logger.Warn().Err(errClose).Msg("unable to close connection properly")
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if err == os.ErrClosed {
				return nil
			}

			return fmt.Errorf("accepting connection: %w", err)
		}

		connLogger := logger.With().Str("remote_addr", conn.RemoteAddr().String()).Logger()

		go handleConnection(ctx, conn, connLogger, h)
	}
}

func PrepareHandlers(tgClient telegram.Client) map[MessageKind]Handler {
	h := map[MessageKind]Handler{
		MessageKindUnknown: nopHandler{},
		MessageKindLog:     logMessageHandler{},
		MessageKindNotify:  notifyTelegramHandler{client: tgClient},
		MessageKindOK:      okHandler{},
		MessageKindFail:    failureHandler{},
	}

	return h
}

// Handler handles incoming request and it's handler's responsobility to answer
// back to the other side.
type Handler interface {
	handle(ctx context.Context, data []byte, w *gob.Encoder) (err error)
}

func handleConnection(ctx context.Context, conn net.Conn, logger zerolog.Logger, handlers map[MessageKind]Handler) {
	go func() {
		<-ctx.Done()
		errClose := conn.Close()
		if errClose != nil {
			logger.Warn().Err(errClose).Msg("unagle to close connection properly")
		}
	}()

	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	var (
		msg Message
		err error
	)

	for {
		err = dec.Decode(&msg)
		if err != nil {
			if err != io.EOF {
				logger.Warn().Err(err).Msg("unable to decode into message")
				return
			}

			logger.Debug().Msg("connection closed")

			return
		}

		if len(msg.RequestID) == 0 {
			msg.RequestID = uuid.New().String()
		}

		msgLogger := logger.With().Str("request_id", msg.RequestID).Logger()

		msgCtx := fcontext.WithRequestID(ctx, msg.RequestID)
		msgCtx = msgLogger.WithContext(msgCtx)

		msgLogger.Debug().
			Str("remote_addr", conn.RemoteAddr().String()).
			Str("kind", msg.Kind.String()).
			Int("len", len(msg.Data)).
			Msg("received message")

		h, ok := handlers[msg.Kind]
		if !ok {
			msgLogger.Warn().Str("kind", msg.Kind.String()).Msg("handled not found")

			err = respondError(ctx, fmt.Sprintf("handler not found for kind: %d", msg.Kind), enc)
			if err != nil {
				log.Printf("unable to write error: %v", err)

				return
			}

			continue
		}

		err = h.handle(msgCtx, msg.Data, enc)
		if err != nil {
			msgLogger.Warn().Err(err).Msg("unable to handle message")

			err = respondError(msgCtx, err.Error(), enc)
		} else {
			err = respondOk(msgCtx, enc)
		}

		if err != nil {
			msgLogger.Warn().Err(err).Msg("responding to client")
		}
	}
}

func sendMessage(ctx context.Context, kind MessageKind, data []byte, r *gob.Encoder) (err error) {
	err = r.Encode(Message{
		RequestID: fcontext.RequestID(ctx),
		Kind:      kind,
		Data:      data,
	})
	if err != nil {
		return fmt.Errorf("encoding message: %w", err)
	}

	return nil
}

func respondError(ctx context.Context, reason string, r *gob.Encoder) (err error) {
	buf := &bytes.Buffer{}

	err = gob.NewEncoder(buf).Encode(failure{Reason: reason})
	if err != nil {
		return fmt.Errorf("encoding data: %w", err)
	}

	err = sendMessage(ctx, MessageKindFail, buf.Bytes(), r)
	if err != nil {
		return fmt.Errorf("encoding data to client: %w", err)
	}

	return nil
}

func respondOk(ctx context.Context, r *gob.Encoder) (err error) {
	err = sendMessage(ctx, MessageKindOK, nil, r)
	if err != nil {
		return fmt.Errorf("encofing data to client: %w", err)
	}

	return nil
}
