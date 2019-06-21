package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

type SendMessageResponse struct {
	MessageID int64 `json:"message_id"`
}

func (api *HTTP) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()
	var rid = fcontext.RequestID(ctx)
	var logger = zerolog.Ctx(ctx)

	var apiKey = r.URL.Query().Get("api")
	var chatID = r.URL.Query().Get("chat_id")
	var text = r.URL.Query().Get("text")

	if len(apiKey) == 0 {
		var response = ServiceError{Message: "api is empty", RequestID: rid}
		asJSON(ctx, w, response, http.StatusBadRequest)
		return
	}

	if len(chatID) == 0 {
		var response = ServiceError{Message: "chat_id is empty", RequestID: rid}
		asJSON(ctx, w, response, http.StatusBadRequest)
		return
	}

	if len(text) == 0 {
		var response = ServiceError{Message: "text is empty", RequestID: rid}
		asJSON(ctx, w, response, http.StatusBadRequest)
		return
	}

	logger.Debug().Str("api_key", apiKey).Str("chat_id", chatID).Str("text", text).Msg("resending to telegram")

	client := http.DefaultClient
	client.Timeout = time.Second * 10

	var requestURL = fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", apiKey)
	request, _ := http.NewRequest(http.MethodGet, requestURL, nil)

	values := request.URL.Query()
	values.Set("chat_id", chatID)
	values.Set("text", text)

	request.URL.RawQuery = values.Encode()

	response, err := client.Do(request)
	if err != nil {
		logger.Error().Err(err).Msg("unable to proceed request")
		ravenRequest := raven.NewHttp(r)
		api.notifier.CaptureError(err, map[string]string{"fn": "handleNextMessage"}, ravenRequest)

		responseError := ServiceError{Message: err.Error(), RequestID: rid}
		asJSON(ctx, w, responseError, http.StatusInternalServerError)
		return
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logger.Error().Err(err).Msg("unable to read body")
		api.notifier.CaptureError(err, nil)

		responseError := ServiceError{Message: err.Error(), RequestID: rid}
		asJSON(ctx, w, responseError, http.StatusInternalServerError)
		return
	}

	// I don't care about error here
	_ = response.Body.Close()

	var telegramResponse SendMessageResponse
	if err := json.Unmarshal(responseData, &telegramResponse); err != nil {
		logger.Error().Err(err).Msg("unable to unmarshal response")
		api.notifier.CaptureError(err, nil)

		responseError := ServiceError{Message: err.Error(), RequestID: rid}
		asJSON(ctx, w, responseError, http.StatusInternalServerError)
		return
	}

	logger.Info().Int64("message_id", telegramResponse.MessageID).Msg("telegram message served")

	w.WriteHeader(http.StatusOK)
}
