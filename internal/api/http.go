package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
	"github.com/rs/zerolog"

	"github.com/ferux/flightcontrolcenter"
	"github.com/ferux/flightcontrolcenter/internal/config"
	"github.com/ferux/flightcontrolcenter/internal/fcontext"
	"github.com/ferux/flightcontrolcenter/internal/yandex"
)

const MaxHeaderBytes = 256 * (1 << 10) // 256 KiB

type HTTP struct {
	srv *http.Server

	yaclient yandex.Client
	logger   zerolog.Logger
	notifier *raven.Client

	requestCount int64
	bootTime     time.Time
}

// NewHTTP prepares new http service
func NewHTTP(cfg config.Application, yaclient yandex.Client, logger zerolog.Logger, nClient *raven.Client) (*HTTP, error) {
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
		logger:   logger,
		bootTime: time.Now(),
		notifier: nClient,
	}
	api.setupRoutes()

	return api, nil
}

func (api *HTTP) setupRoutes() {
	router := mux.NewRouter()
	router.Use(middlewareCounter(api), middlewareRequestID(), middlewareLogger(api.logger, api))
	router.HandleFunc("/info", api.handleInfo)
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/nextbus", api.handleNextBus).Methods(http.MethodGet)

	api.srv.Handler = router
}

// Serve connections
func (api *HTTP) Serve() {
	go func() {
		api.logger.Info().Str("listen", api.srv.Addr).Msg("serving http")
		err := api.srv.ListenAndServe()
		if err != nil {
			api.logger.Error().Err(err).Msg("interrupted")
		}
	}()
}

// Shutdown the server
func (api *HTTP) Shutdown(ctx context.Context) error {
	return api.srv.Shutdown(ctx)
}

func middlewareRequestID() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			rid := r.Header.Get("x-request-id")
			if len(rid) == 0 {
				rid = uuid.New()
			}

			w.Header().Set("x-request-id", rid)
			r = r.WithContext(fcontext.WithRequestID(ctx, rid))

			h.ServeHTTP(w, r)
		})
	}
}

func middlewareLogger(logger zerolog.Logger, api *HTTP) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			rid := fcontext.RequestID(ctx)
			lg := logger.With().Str("request_id", rid).Logger()
			r = r.WithContext(lg.WithContext(ctx))
			start := time.Now()
			lg.Debug().
				Str("method", r.Method).
				Str("request_uri", r.RequestURI).
				Msg("accepted")

			h.ServeHTTP(w, r)

			lg.Info().Str("took", time.Since(start).String()).Msg("served")
		})
	}
}

func middlewareCounter(api *HTTP) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&api.requestCount, 1)
			h.ServeHTTP(w, r)
		})
	}
}

func (api *HTTP) handleNextBus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := zerolog.Ctx(ctx)

	stopID := r.URL.Query().Get("stop_id")
	if len(stopID) == 0 {
		http.Error(w, "empty stop id", http.StatusBadRequest)
		logger.Error().Msg("stop_id is empty")
		return
	}

	transport, err := api.yaclient.Fetch(stopID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Error().Err(err).Msg("fetching stop id")
		return
	}

	var first = yandex.TransportInfo{Arrive: time.Now().Add(time.Hour * 24 * 7)}
	var filterRoute = r.URL.Query().Get("route")
	for _, tr := range transport.IncomingTransport {
		if !strings.Contains(tr.Name, filterRoute) {
			continue
		}

		if tr.Arrive.Before(first.Arrive) {
			first = tr
		}
	}

	var response = struct {
		Name string
		Next string
	}{
		Name: first.Name,
		Next: first.Arrive.Format("15:04"),
	}

	asJSON(ctx, w, &response, http.StatusOK)
}

func (api *HTTP) handleInfo(w http.ResponseWriter, r *http.Request) {
	var response = struct {
		Revision     string    `json:"revision"`
		Branch       string    `json:"branch"`
		Boot         time.Time `json:"boot"`
		Uptime       string    `json:"uptime"`
		RequestCount int64     `json:"request_count"`
	}{
		Revision:     flightcontrolcenter.Revision,
		Branch:       flightcontrolcenter.Branch,
		Boot:         api.bootTime,
		Uptime:       time.Since(api.bootTime).String(),
		RequestCount: api.requestCount,
	}

	asJSON(r.Context(), w, &response, http.StatusOK)
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
