package fccgob

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/ferux/flightcontrolcenter/internal/telegram"

	"github.com/rs/zerolog"
)

type notifyTelegram struct {
	APIKey  string
	ChatID  string
	Message string
}

type notifyTelegramHandler struct {
	client telegram.Client
}

func (h notifyTelegramHandler) handle(ctx context.Context, data []byte, _ *gob.Encoder) (err error) {
	var msg notifyTelegram

	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&msg)
	if err != nil {
		return fmt.Errorf("decoding data: %w", err)
	}

	err = h.client.SendMessageViaHTTP(ctx, msg.APIKey, msg.ChatID, msg.Message)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

type logMessage struct {
	Text     string
	Severity logSeverity
}

type logMessageHandler struct{}

func (logMessageHandler) handle(ctx context.Context, data []byte, _ *gob.Encoder) (err error) {
	var msg logMessage

	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&msg)
	if err != nil {
		return fmt.Errorf("decoding data: %w", err)
	}

	level := zerolog.Level(uint8(msg.Severity))
	zerolog.Ctx(ctx).WithLevel(level).Str("text", msg.Text).Msg("logging event")

	return nil
}

type logSeverity uint8

const (
	LogSeverityDebu = iota
	LogSeverityInfo
	LogSeverityWarn
	LogSeverityErro
)

type ok struct{}

type okHandler struct{}

func (h okHandler) handle(_ context.Context, _ []byte, _ *gob.Encoder) (err error) {
	// do nothing
	return nil
}

type failure struct {
	Reason string
}

type failureHandler struct{}

func (h failureHandler) handle(ctx context.Context, data []byte, _ *gob.Encoder) (err error) {
	var msg failure

	err = gob.NewDecoder(bytes.NewReader(data)).Decode(&msg)
	if err != nil {
		return fmt.Errorf("decoding data: %w", err)
	}

	zerolog.Ctx(ctx).Warn().Str("reason", msg.Reason).Msg("request error")

	return nil
}

type nopHandler struct{}

func (nopHandler) handle(_ context.Context, _ []byte, _ *gob.Encoder) error {
	return nil
}
