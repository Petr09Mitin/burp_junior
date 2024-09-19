package routers

import (
	"net/http"

	"github.com/burp_junior/internal/rest/api"
	"github.com/burp_junior/internal/rest/proxy"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func MountProxyRouter(logger *zap.Logger) {
	r := mux.NewRouter()

	r.HandleFunc("/", proxy.ProxyHandler)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Error("Proxy failed to listen: ", zap.Error(err))
	}
}

func MountAPIRouter(logger *zap.Logger) {
	r := mux.NewRouter()

	r.HandleFunc("/requests", api.APIHandler)

	err := http.ListenAndServe(":8000", r)
	if err != nil {
		logger.Error("WebAPI failed to listen: ", zap.Error(err))
	}
}
