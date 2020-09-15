package telegram

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/valyala/fastjson"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/model"
)

// Client for interacting with telegram.
type Client interface {
	SendMessageViaHTTP(ctx context.Context, apiKey, chatID, text string) error
}

type client struct {
	c *http.Client
}

// New creates new telegram client.
func New() Client {
	c := http.DefaultClient
	c.Timeout = time.Second * 10

	return &client{c: c}
}

func (client *client) SendMessageViaHTTP(ctx context.Context, apiKey, chatID, text string) (err error) {
	logger := zerolog.Ctx(ctx).With().Str("pkg", "telegram").Logger()
	rid := fcontext.RequestID(ctx)

	if len(apiKey) == 0 {
		return model.ServiceError{Message: "api is empty", RequestID: rid, Code: http.StatusBadRequest}
	}

	if len(chatID) == 0 {
		return model.ServiceError{Message: "chat_id is empty", RequestID: rid}
	}

	if len(text) == 0 {
		return model.ServiceError{Message: "text is empty", RequestID: rid}
	}

	logger.Debug().Str("api_key", apiKey).Str("chat_id", chatID).Str("text", text).Msg("resending to telegram")

	requestURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", apiKey)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)

	values := request.URL.Query()
	values.Set("chat_id", chatID)
	values.Set("text", text)

	request.URL.RawQuery = values.Encode()

	response, err := client.c.Do(request)
	if err != nil {
		return model.ServiceError{Message: err.Error(), RequestID: rid, Code: http.StatusInternalServerError}
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return model.ServiceError{Message: err.Error(), RequestID: rid, Code: http.StatusInternalServerError}
	}

	logger.Debug().RawJSON("response", responseData).Msg("accepted message")

	// I don't care about error here
	_ = response.Body.Close()

	v, err := fastjson.ParseBytes(responseData)
	if err != nil {
		logger.Error().Err(err).Msg("unable to parse response")

		return model.ServiceError{Message: err.Error(), RequestID: rid, Code: http.StatusInternalServerError}
	}

	logger.Info().Int64("message_id", int64(v.GetInt("message_id"))).Msg("response from telegram")

	return nil
}
