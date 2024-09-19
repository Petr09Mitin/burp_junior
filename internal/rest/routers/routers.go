package routers

import (
	"net/http"

	rest_api "github.com/burp_junior/internal/rest/api"
	rest_proxy "github.com/burp_junior/internal/rest/proxy"
	"github.com/burp_junior/usecase/proxy"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func MountProxyRouter(logger *zap.Logger) {
	r := mux.NewRouter()

	proxyHandler := rest_proxy.NewProxyHandler(proxy.NewProxyService())

	r.HandleFunc("/", proxyHandler.HandleProxy)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		logger.Error("Proxy failed to listen: ", zap.Error(err))
	}
}

func MountAPIRouter(logger *zap.Logger) {
	r := mux.NewRouter()

	r.HandleFunc("/requests", rest_api.APIHandler)

	err := http.ListenAndServe(":8000", r)
	if err != nil {
		logger.Error("WebAPI failed to listen: ", zap.Error(err))
	}
}
