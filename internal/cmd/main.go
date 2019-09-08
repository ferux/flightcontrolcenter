package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/api"
	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/ping"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	pkglogger "github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var (
	revision string // nolint:gochecknoglobals
	branch   string // nolint:gochecknoglobals
	env      string // nolint:gochecknoglobals
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
			Str("revision", revision).
			Str("branch", branch).
			Str("env", env).
			Msg("parsing config file")
	}

	if *showRevision {
		return
	}

	logger.
		Debug().
		Interface("config", cfg).
		Str("rev", revision).
		Str("branch", branch).
		Str("env", env).
		Msg("starting application")

	yaclient, err := yandex.New(pkglogger.New())
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create yandex client")
	}

	// TODO: hide sentry under interface implementation
	notifierClient, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:         cfg.SentryDSN,
		Debug:       !strings.EqualFold(env, "production"),
		Environment: env,
		ServerName:  "fcc.loyso.art",
		Release:     revision,
		SampleRate:  1,
	})

	// notifierClient, err := raven.New(cfg.SentryDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create sentry client")
	}

	tgclient := telegram.New()
	var appInfo = model.ApplicationInfo{
		Branch:      branch,
		Revision:    revision,
		Environment: env,
	}

	dstore := ping.New(notifierClient)
	dstore.Subscribe(deviceStateNotify(tgclient, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID))

	api, _ := api.NewHTTP(cfg, yaclient, tgclient, dstore, logger, notifierClient, appInfo)
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
	message.WriteString(branch)
	message.WriteString(" env=")
	message.WriteString(env)
	message.WriteString(" revision=")
	message.WriteString(revision)
	return tgclient.SendMessageViaHTTP(ctx, api, chatID, message.String())
}

func deviceStateNotify(tgclient telegram.Client, api, chatID string) ping.NotifyDeviceStateChanged {
	return func(d ping.Device) {
		var message = strings.Builder{}
		message.Grow(128)
		message.WriteString(d.Name)
		message.WriteString(" [")
		message.WriteString(d.Type)
		message.WriteString("] @ ")
		message.WriteString(d.IP)
		message.WriteString("device ")
		message.WriteString(d.Name)
		message.WriteString(" (")
		message.WriteString(d.IP)
		if d.IsOnline {
			message.WriteString(") is online now")
		} else {
			message.WriteString(") has gone offline")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		_ = tgclient.SendMessageViaHTTP(ctx, api, chatID, message.String())
	}
}
