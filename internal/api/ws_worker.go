package api

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/ping"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

func (api *HTTP) handleWS() http.Handler {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: time.Second * 5,
		ReadBufferSize:   4 << 10, // 4 KiB
		WriteBufferSize:  4 << 10, // 4 KiB
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var msg ping.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			var response = model.ServiceError{
				Message:   "unable to unmarshal message",
				RequestID: fcontext.RequestID(ctx),
				Code:      http.StatusBadRequest,
			}

			api.serveError(ctx, w, r, response)
			return
		}

		addr := r.Header.Get("X-Forwarded-For")
		if len(addr) == 0 {
			addr, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		msg.IP = addr

		api.dstore.Ping(msg)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			var response = model.ServiceError{
				Message:   "unable to upgrade to websockets",
				RequestID: fcontext.RequestID(ctx),
				Code:      http.StatusBadRequest,
			}

			api.serveError(ctx, w, r, response)
			return
		}

		wsconn := wsConnection{
			conn:     conn,
			deviceID: msg.ID,
			realIP:   addr,
		}

		go handleWSConnection(wsconn, api.logger, api.dstore)
	})
}

type wsConnection struct {
	conn     *websocket.Conn
	deviceID string
	realIP   string
}

const pongWait = time.Second * 15

func handleWSConnection(ws wsConnection, logger zerolog.Logger, store ping.Store) {
	defer func() {
		store.MarkOffline(ws.deviceID)
		logger.Debug().Str("device_id", ws.deviceID).Msg("has gone offline")
	}()

	var err error
	var p []byte

	for ; err == nil; _, p, err = ws.conn.ReadMessage() {
		logger.Debug().RawJSON("accepted", p).Msg("message from device")
	}
}
