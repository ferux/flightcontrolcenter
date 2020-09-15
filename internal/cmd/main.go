package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"

	"github.com/ferux/flightcontrolcenter/internal/api/fccgob"
	"github.com/ferux/flightcontrolcenter/internal/api/fcchttp"
	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/dnsupdater"
	pkglogger "github.com/ferux/flightcontrolcenter/internal/logger"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/ping"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"
)

var (
	revision string // nolint:gochecknoglobals
	branch   string // nolint:gochecknoglobals
	env      string // nolint:gochecknoglobals
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.New(os.Stdout)
	path := flag.String("config", "./config.json", "path to config")
	showRevision := flag.Bool("revision", false, "show version of the application")

	flag.Parse()

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
	if err != nil {
		logger.Fatal().Err(err).Msg("can't create sentry client")
	}

	var tgclient telegram.Client = telegram.Mock{}

	if cfg.NotifyTelegram != (config.NotifyTelegram{}) {
		tgclient = telegram.New()
	}

	appInfo := model.ApplicationInfo{
		Branch:      branch,
		Revision:    revision,
		Environment: env,
	}

	dstore := ping.New(notifierClient)
	dstore.Subscribe(deviceStateNotify(tgclient, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID))

	go func() {
		if cfg.GOBAPI == nil {
			logger.Info().Str("reason", "no settings").Msg("not starting gob api")

			return
		}

		logger.Info().Str("listen", cfg.GOBAPI.Listen).Msg("running gob api")

		handlers := fccgob.PrepareHandlers(tgclient)

		errGOB := fccgob.Serve(ctx, *cfg.GOBAPI, logger, handlers)
		if errGOB != nil {
			logger.Error().Err(errGOB).Msg("unable to start gob")
		}
	}()

	dns := dnsupdater.New(context.Background(), cfg.DNSUpdater)

	if cfg.HTTP != nil {
		httpapi, err := fcchttp.NewHTTP(*cfg.HTTP, yaclient, tgclient, dns, dstore, logger, notifierClient, appInfo)
		if err != nil {
			logger.Error().Err(err).Msg("running http client")
		}

		httpapi.Serve()
	} else {
		logger.Info().Str("reason", "no settings").Msg("not starting http api")
	}

	go func(ctx context.Context, tgclient telegram.Client, api, chatID string, logger zerolog.Logger) {
		if err := sendNotificationMessage(ctx, tgclient, api, chatID); err != nil {
			logger.Error().Err(err).Msg("can't notify telegram")
		}
	}(ctx, tgclient, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID, logger)

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGQUIT)
	<-s
	cancel()

	errNotify := tgclient.SendMessageViaHTTP(ctx, cfg.NotifyTelegram.API, cfg.NotifyTelegram.ChatID, "shutting down")
	if errNotify != nil {
		logger.Error().Err(errNotify).Msg("error notifying via tg")
	}
}

func sendNotificationMessage(ctx context.Context, tgclient telegram.Client, api, chatID string) error {
	message := strings.Builder{}

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
		message := strings.Builder{}

		message.Grow(128)
		message.WriteString(d.Name)
		message.WriteString(" [")
		message.WriteString(d.Type)
		message.WriteString("] @ ")
		message.WriteString(d.IP)

		if d.IsOnline {
			message.WriteString(" is online now")
		} else {
			message.WriteString(" has gone offline")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		_ = tgclient.SendMessageViaHTTP(ctx, api, chatID, message.String())
	}
}
