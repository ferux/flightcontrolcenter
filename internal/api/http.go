package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/raven-go"
	"github.com/rs/zerolog"
)

const MaxHeaderBytes = 256 * (1 << 10) // 256 KiB

type HTTP struct {
	srv *http.Server

	yaclient yandex.Client
	tgclient telegram.Client
	logger   zerolog.Logger
	notifier *raven.Client

	requestCount int64
	bootTime     time.Time
}

// NewHTTP prepares new http service
func NewHTTP(cfg config.Application, yaclient yandex.Client, tgclient telegram.Client, logger zerolog.Logger, nClient *raven.Client) (*HTTP, error) {
	to := cfg.HTTP.Timeout.Std()
	srv := &http.Server{
		Addr:              cfg.HTTP.Listen,
		ReadTimeout:       to,
		ReadHeaderTimeout: to,
		WriteTimeout:      to,
		IdleTimeout:       to,
		TLSConfig:         &tls.Config{InsecureSkipVerify: true},
		MaxHeaderBytes:    MaxHeaderBytes,
	}

	api := &HTTP{
		srv:      srv,
		yaclient: yaclient,
		tgclient: tgclient,
		logger:   logger,
		bootTime: time.Now(),
		notifier: nClient,
	}
	api.setupRoutes()

	return api, nil
}

// Serve connections
func (api *HTTP) Serve() {
	go func() {
		api.logger.Info().Str("listen", api.srv.Addr).Msg("serving http")
		err := api.srv.ListenAndServe()
		if err != nil {
			api.logger.Error().Err(err).Msg("interrupted")
			api.notifier.CaptureError(err, map[string]string{"msg": "interrupted"})
		}
	}()
}

// Shutdown the server
func (api *HTTP) Shutdown(ctx context.Context) error {
	return api.srv.Shutdown(ctx)
}

func asJSON(ctx context.Context, w http.ResponseWriter, obj interface{}, code int) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		logger := zerolog.Ctx(ctx)
		logger.Error().Err(err).Msg("encoding json")
	}
}
