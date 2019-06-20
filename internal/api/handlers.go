package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/raven-go"
	"github.com/rs/zerolog"
)

func (api *HTTP) handleNextBus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rid := fcontext.RequestID(ctx)
	logger := zerolog.Ctx(ctx)

	stopID := r.URL.Query().Get("stop_id")
	if len(stopID) == 0 {
		var response = ServiceError{
			Message:   "stop_id is empty",
			RequestID: rid,
		}
		asJSON(ctx, w, &response, http.StatusUnprocessableEntity)
		logger.Error().Msg("stop_id is empty")
		rReq := raven.NewHttp(r)
		api.notifier.CaptureError(errors.New("stop_id is empty"), nil, rReq)
		return
	}

	transport, err := api.yaclient.Fetch(stopID)
	if err != nil {
		var response = ServiceError{
			Message:   "something went wrong",
			RequestID: fcontext.RequestID(ctx),
		}
		asJSON(ctx, w, &response, http.StatusInternalServerError)
		logger.Error().Err(err).Msg("fetching stop id")
		rReq := raven.NewHttp(r)
		api.notifier.CaptureError(err, nil, rReq)
		return
	}

	if len(transport.IncomingTransport) == 0 {
		var response = ServiceError{Message: "not found", RequestID: rid}
		asJSON(ctx, w, &response, http.StatusNotFound)
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

	var response = StopInfo{
		Name: first.Name,
		Next: first.Arrive.Format("15:04"),
	}

	asJSON(ctx, w, &response, http.StatusOK)
}

type ServiceInfo struct {
	Revision     string    `json:"revision"`
	Branch       string    `json:"branch"`
	Boot         time.Time `json:"boot"`
	Uptime       string    `json:"uptime"`
	RequestCount int64     `json:"request_count"`
}

func (api *HTTP) handleInfo(w http.ResponseWriter, r *http.Request) {
	var response = ServiceInfo{
		Revision:     flightcontrolcenter.Revision,
		Branch:       flightcontrolcenter.Branch,
		Boot:         api.bootTime,
		Uptime:       time.Since(api.bootTime).String(),
		RequestCount: api.requestCount,
	}

	asJSON(r.Context(), w, &response, http.StatusOK)
}
