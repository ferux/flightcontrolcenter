package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/api"
	"github.com/ferux/flightcontrolcenter/internal/config"
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
			Msg("parsing config file")
	}

	logger.Debug().Interface("config", cfg).Str("rev", flightcontrolcenter.Branch).Str("branch", flightcontrolcenter.Branch).Msg("starting application")

	client, err := yandex.New(nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create yandex client: ")
	}

	api, _ := api.NewHTTP(cfg, client, logger)
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
