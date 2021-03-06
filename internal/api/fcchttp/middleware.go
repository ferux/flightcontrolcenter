package fcchttp

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/pborman/uuid"
	"github.com/rs/zerolog"

	"github.com/ferux/flightcontrolcenter/internal/fcontext"
)

func middlewareCORS() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == http.MethodOptions {
				return
			}

			h.ServeHTTP(w, r)
		})
	}
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

func middlewareLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
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
				Str("remote_addr", r.RemoteAddr).
				Str("remote_addr_header", r.Header.Get("X-Forwarded-For")).
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
