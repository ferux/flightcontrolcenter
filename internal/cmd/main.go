package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/rs/zerolog"

	"github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/api"
	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"
)

func main() {
	path := flag.String("config", "./config.json", "path to config")
	showRevision := flag.Bool("revision", false, "show version of the application")

	flag.Parse()

	if *showRevision {
		fmt.Println(flightcontrolcenter.Revision)
		return
	}

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

	go func() {
		if err := sendNotificationMessage(ctx, tgclient, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID); err != nil {
			logger.Error().Err(err).Msg("can't notify telegram")
		}
	}()

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
	var b = flightcontrolcenter.Branch
	var e = flightcontrolcenter.Env
	var r = flightcontrolcenter.Revision
	message := fmt.Sprintf("fcc branch=%s env=%s revision=%s", b, e, r)
	return tgclient.SendMessageViaHTTP(ctx, api, chatID, message)
}
