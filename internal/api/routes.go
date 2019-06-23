package api

import (
	"net/http"

	"github.com/ferux/flightcontrolcenter/internal/static"

	"github.com/gorilla/mux"
)

func (api *HTTP) setupRoutes() {
	router := mux.NewRouter()

	// swagger files
	router.Handle("/swagger", http.RedirectHandler("/swagger/", http.StatusMovedPermanently))
	router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(static.AssetFile())))

	// api/v1 base path handlers
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Use(middlewareCounter(api), middlewareRequestID(), middlewareLogger(api.logger))
	v1.HandleFunc("/info", api.handleInfo)
	v1.HandleFunc("/nextbus", api.handleNextBus).Methods(http.MethodGet)
	v1.HandleFunc("/send_message", api.handleSendMessage).Methods(http.MethodGet)
	api.srv.Handler = router
}
