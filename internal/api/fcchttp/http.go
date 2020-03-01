package fcchttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/ping"
	"github.com/ferux/flightcontrolcenter/internal/telegram"
	"github.com/ferux/flightcontrolcenter/internal/yandex"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

const (
	maxHeaderBytes = 256 * (1 << 10) // 256 KiB
	contentType    = "content-type"
	contentJSON    = "application/json"
)

type HTTP struct {
	srv *http.Server

	dstore   ping.Store
	yaclient yandex.Client
	tgclient telegram.Client
	logger   zerolog.Logger
	notifier *sentry.Client

	requestCount int64
	bootTime     time.Time
}

// NewHTTP prepares new http service
func NewHTTP(
	cfg config.HTTP,
	yaclient yandex.Client,
	tgclient telegram.Client,
	dstore ping.Store,
	logger zerolog.Logger,
	nClient *sentry.Client,
	appInfo model.ApplicationInfo,
) (*HTTP, error) {
	to := cfg.Timeout.Std()
	srv := &http.Server{
		Addr:              cfg.Listen,
		ReadTimeout:       to,
		ReadHeaderTimeout: to,
		WriteTimeout:      to,
		IdleTimeout:       to,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	api := &HTTP{
		srv:      srv,
		yaclient: yaclient,
		tgclient: tgclient,
		dstore:   dstore,
		logger:   logger,
		bootTime: time.Now(),
		notifier: nClient,
	}
	api.setupRoutes(appInfo)

	return api, nil
}

// Serve connections
func (api *HTTP) Serve() {
	go func() {
		api.logger.Info().Str("listen", api.srv.Addr).Msg("serving http")
		err := api.srv.ListenAndServe()
		if err != nil {
			api.logger.Error().Err(err).Msg("interrupted")
			api.notifier.CaptureException(err, nil, sentry.NewScope())
		}
	}()
}

// Shutdown the server
func (api *HTTP) Shutdown(ctx context.Context) error {
	return api.srv.Shutdown(ctx)
}

func asJSON(ctx context.Context, w http.ResponseWriter, obj interface{}, code int) {
	w.Header().Set(contentType, contentJSON)
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("encoding json")
	}
}
