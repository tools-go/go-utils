package server

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/tools-go/go-utils/middleware"
	"github.com/tools-go/go-utils/trace"
)

type healthz struct{}

// Healthz controller example
var Healthz Controller = &healthz{}

func (h *healthz) Register(router *mux.Router) {
	subrouter := router.Path("/healthz").Subrouter()
	subrouter.Methods("GET").HandlerFunc(middleware.RecoverWithTrace("healthcheck").HandlerFunc(h.check))
}

func (h *healthz) check(w http.ResponseWriter, req *http.Request) {
	tracer := trace.GetTraceFromRequest(req)
	tracer.Info("check ok")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
