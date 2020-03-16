package fcchttp

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/ping"
	"github.com/ferux/flightcontrolcenter/internal/templates"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

const forwardedForHeader = "X-Forwarded-For"

func (api *HTTP) handleNextBus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := fcontext.RequestID(ctx)
	logger := zerolog.Ctx(ctx).With().Str("fn", "handleNextBus").Logger()

	stopID := r.URL.Query().Get("stop_id")
	if len(stopID) == 0 {
		var response = model.ServiceError{
			Message:   "stop_id is empty",
			RequestID: rid,
			Code:      http.StatusUnprocessableEntity,
		}

		api.serveError(ctx, w, r, response)

		return
	}

	var prognosis bool
	if r.URL.Query().Get("prognosis") == "true" {
		prognosis = true
	}

	logger.Debug().Bool("prognosis", prognosis).Str("stop_id", stopID).Msg("fetching")

	transportAll, err := api.yaclient.Fetch(ctx, stopID, prognosis)
	if err != nil {
		var response = model.ServiceError{
			Message:   "something went wrong",
			RequestID: fcontext.RequestID(ctx),
			Code:      http.StatusInternalServerError,
		}

		api.serveError(ctx, w, r, response)

		return
	}

	var filterRoute = r.URL.Query().Get("route")

	var transports = make([]yandex.TransportInfo, 0, len(transportAll.IncomingTransport))

	for _, tr := range transportAll.IncomingTransport {
		if !strings.Contains(tr.Name, filterRoute) {
			continue
		}

		transports = append(transports, tr)
	}

	logger.Debug().Interface("transport", transports).Msg("done")

	var first = yandex.TransportInfo{Arrive: time.Now().Add(time.Hour * 24)}

	var found bool

	switch len(transports) {
	case 0:
		logger.Debug().Msg("no incoming transport")

		var response = model.ServiceError{
			Message:   "not found",
			RequestID: rid,
			Code:      http.StatusNotFound,
		}

		api.serveError(ctx, w, r, response)

		return
	case 1:
		first = transports[0]
		found = true
	default:
		logger.Debug().Int("amount", len(transports)).Msg("found buses")

		first = transports[0]

		for _, tr := range transports {
			if !strings.Contains(tr.Name, filterRoute) {
				continue
			}

			if tr.Arrive.Hour() < time.Now().Hour() {
				continue
			}

			if tr.Arrive.Before(first.Arrive) {
				found = true
				first = tr
			}
		}
	}

	logger.Debug().Interface("result", first).Msg("done")

	if !strings.Contains(first.Name, filterRoute) && !found {
		var response = model.ServiceError{
			Message:   "not found",
			RequestID: rid,
			Code:      http.StatusNotFound,
		}

		api.serveError(ctx, w, r, response)

		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	(&templates.NextBus{
		Name:      first.Name,
		Next:      first.Arrive.Format("15:04"),
		Method:    first.Method,
		RequestID: rid,
	}).WriteJSON(w)
}

func (api *HTTP) handleInfo(info model.ApplicationInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)

		(&templates.MarshalData{
			Revision:     info.Revision,
			Branch:       info.Branch,
			Environment:  info.Environment,
			BootTime:     api.bootTime.String(),
			Uptime:       time.Since(api.bootTime).Seconds(),
			RequestCount: int(api.requestCount),
		}).WriteJSON(w)
	}
}

func (api *HTTP) handleSendMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		var apiKey = r.URL.Query().Get("api")
		var chatID = r.URL.Query().Get("chat_id")
		var text = r.URL.Query().Get("text")

		err := api.tgclient.SendMessageViaHTTP(ctx, apiKey, chatID, text)
		if err != nil {
			api.serveError(ctx, w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (api *HTTP) handlePingMessage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		var msg ping.Message
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			api.serveError(ctx, w, r, err)
			return
		}

		if len(msg.ID) != 64 {
			api.logger.Warn().Msg("device without id")
			err = model.ServiceError{
				Code:      http.StatusBadRequest,
				RequestID: fcontext.RequestID(ctx),
				Message:   "empty id",
			}
			api.serveError(ctx, w, r, err)
			return
		}

		addr := r.Header.Get(forwardedForHeader)
		if len(addr) == 0 {
			addr, _, _ = net.SplitHostPort(r.RemoteAddr)
		}

		msg.IP = addr
		api.logger.Debug().Interface("msg", msg).Msg("served")
		api.dstore.Ping(msg)
	}
}

func (api *HTTP) handleGetDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		devices := api.dstore.GetDevices()

		asJSON(ctx, w, devices, http.StatusOK)
	}
}

func (api *HTTP) serveError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var (
		logger     = zerolog.Ctx(ctx)
		rid        = fcontext.RequestID(ctx)
		eventLevel = sentry.LevelFatal

		responseError model.ServiceError
	)

	switch terr := err.(type) {
	case model.ServiceError:
		responseError = terr
		if terr.Code == 0 {
			responseError.Code = http.StatusInternalServerError
		}
	default:
		responseError.Code = http.StatusInternalServerError
		responseError.Message = err.Error()
		responseError.RequestID = rid
	}

	if responseError.Code != http.StatusInternalServerError {
		eventLevel = sentry.LevelError
	}

	logger.Error().Err(responseError).Msg("captured error")

	st := sentry.NewStacktrace()

	event := sentry.NewEvent()

	event.Exception = []sentry.Exception{{Stacktrace: st}}
	event.Message = responseError.Message
	event.Environment = "production"
	event.Level = eventLevel
	event.Contexts["request_id"] = rid
	event.Request = event.Request.FromHTTPRequest(r)

	api.notifier.CaptureEvent(event, &sentry.EventHint{
		OriginalException: err,
	}, sentry.NewScope())

	asJSON(ctx, w, responseError, responseError.Code)
}
