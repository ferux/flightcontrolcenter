package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/api"
	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/raven-go"
	"github.com/rs/zerolog"
)

func main() {
	path := flag.String("config", "./config.json", "path to config")
	showRevision := flag.Bool("revision", false, "show version of the application")

	flag.Parse()

	logger := zerolog.New(os.Stdout)
	cfg, err := config.Parse(*path)
	if err != nil {
		logger.
			Fatal().
			Err(err).
			Str("revision", flightcontrolcenter.Revision).
			Str("branch", flightcontrolcenter.Branch).
			Str("env", flightcontrolcenter.Env).
			Msg("parsing config file")
	}

	if *showRevision == true {
		return
	}

	logger.
		Debug().
		Interface("config", cfg).
		Str("rev", flightcontrolcenter.Branch).
		Str("branch", flightcontrolcenter.Branch).
		Msg("starting application")

	yaclient, err := yandex.New(nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create yandex client")
	}

	// TODO: hide sentry under interface implementation
	notifierClient, err := raven.New(cfg.SentryDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create sentry client")
	}
	notifierClient.SetRelease(flightcontrolcenter.Revision)
	notifierClient.SetEnvironment(flightcontrolcenter.Env)

	tgclient := telegram.New()
	api, _ := api.NewHTTP(cfg, yaclient, tgclient, logger, notifierClient)
	api.Serve()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	go func(ctx context.Context, tgclient telegram.Client, api, chatID string, logger zerolog.Logger) {
		if err := sendNotificationMessage(ctx, tgclient, api, chatID); err != nil {
			logger.Error().Err(err).Msg("can't notify telegram")
		}
	}(ctx, tgclient, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID, logger)

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGQUIT)
	<-s

	errNotify := tgclient.SendMessageViaHTTP(ctx, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID, "shutting down")
	if errNotify != nil {
		logger.Error().Err(errNotify).Msg("error notifying via tg")
	}

	if errShut := api.Shutdown(ctx); errShut != nil {
		logger.Error().Err(errShut).Msg("error shutting down server")
	}
}

func sendNotificationMessage(ctx context.Context, tgclient telegram.Client, api, chatID string) error {
	var message = strings.Builder{}
	message.Grow(64)
	message.WriteString("fcc branch=")
	message.WriteString(flightcontrolcenter.Branch)
	message.WriteString(" env=")
	message.WriteString(flightcontrolcenter.Env)
	message.WriteString(" revision=")
	message.WriteString(flightcontrolcenter.Revision)
	return tgclient.SendMessageViaHTTP(ctx, api, chatID, message.String())
}
