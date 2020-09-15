package fcchttp

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ferux/flightcontrolcenter/internal/model"
	"github.com/ferux/flightcontrolcenter/internal/static"
)

func (api *HTTP) setupRoutes(info model.ApplicationInfo) {
	router := mux.NewRouter()

	router.Use(middlewareCORS())

	// swagger files
	router.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))
	router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(static.AssetFile())))

	// api/v1 base path handlers
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Use(middlewareCounter(api), middlewareRequestID(), middlewareLogger(api.logger))
	v1.HandleFunc("/info", api.handleInfo(info))
	v1.HandleFunc("/nextbus", api.handleNextBus).Methods(http.MethodGet)
	v1.HandleFunc("/send_message", api.handleSendMessage()).Methods(http.MethodGet)
	v1.HandleFunc("/ping", api.handlePingMessage()).Methods(http.MethodPost)
	v1.HandleFunc("/devices", api.handleGetDevices()).Methods(http.MethodGet)
	// Use GET here because my router does not support POST methods in easy way.
	v1.HandleFunc("/update_dns", api.handleDNSUpdate()).Methods(http.MethodGet)
	api.srv.Handler = router
}
