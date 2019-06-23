package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	fcc "github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/templates"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/raven-go"
	"github.com/rs/zerolog"
)

func (api *HTTP) handleNextBus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := fcontext.RequestID(ctx)

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

	transport, err := api.yaclient.Fetch(stopID)
	if err != nil {
		var response = model.ServiceError{
			Message:   "something went wrong",
			RequestID: fcontext.RequestID(ctx),
			Code:      http.StatusInternalServerError,
		}

		api.serveError(ctx, w, r, response)
		return
	}

	if len(transport.IncomingTransport) == 0 {
		var response = model.ServiceError{
			Message:   "not found",
			RequestID: rid,
			Code:      http.StatusNotFound,
		}

		api.serveError(ctx, w, r, response)
		return
	}

	var first = yandex.TransportInfo{Arrive: time.Now().Add(time.Hour * 24 * 7)}
	var filterRoute = r.URL.Query().Get("route")
	for _, tr := range transport.IncomingTransport {
		if !strings.Contains(tr.Name, filterRoute) {
			continue
		}

		// let's think the next bus time can't be after 23 hours.
		if tr.Arrive.Hour() < time.Now().Hour() {
			tr.Arrive = tr.Arrive.Add(time.Hour * 24)
		}

		if tr.Arrive.Before(first.Arrive) {
			first = tr
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	(&templates.NextBus{
		Name: first.Name,
		Next: first.Arrive.Format("15:04"),
	}).WriteJSON(w)
}

func (api *HTTP) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	(&templates.MarshalData{
		Revision:     fcc.Revision,
		Branch:       fcc.Branch,
		BootTime:     api.bootTime.String(),
		Uptime:       time.Since(api.bootTime).Seconds(),
		RequestCount: int(api.requestCount),
	}).WriteJSON(w)
}

func (api *HTTP) handleSendMessage(w http.ResponseWriter, r *http.Request) {
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

func (api *HTTP) serveError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	var logger = zerolog.Ctx(ctx)
	var rid = fcontext.RequestID(ctx)

	var responseError model.ServiceError
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

	logger.Error().Err(responseError).Msg("captured error")

	ravenRequest := raven.NewHttp(r)
	api.notifier.CaptureError(err, nil, ravenRequest)

	asJSON(ctx, w, responseError, responseError.Code)
}
