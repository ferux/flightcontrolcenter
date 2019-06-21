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

	logger.Debug().Interface("config", cfg).Str("rev", flightcontrolcenter.Branch).Str("branch", flightcontrolcenter.Branch).Msg("starting application")

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

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGQUIT)
	<-s

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	if errShut := api.Shutdown(ctx); errShut != nil {
		logger.Error().Err(errShut).Msg("error shuting down server: ")
	}
}
