package routers

import (
	"log"
	"net/http"

	rest_api "github.com/burp_junior/internal/rest/api"
	rest_proxy "github.com/burp_junior/internal/rest/proxy"
	"github.com/gorilla/mux"
)

func MountProxyRouter(rs rest_proxy.RequestService) {
	proxyHandler := rest_proxy.NewProxyHandler(rs)

	proxyPort := ":8080"
	log.Println("Proxy is running on port " + proxyPort)
	err := http.ListenAndServe(proxyPort, proxyHandler)
	if err != nil {
		log.Println("Proxy failed to listen: ", err)
		return
	}

	return
}

func MountAPIRouter(rs rest_api.RequestService) {
	r := mux.NewRouter()

	h := rest_api.NewAPIHandler(rs)

	r.HandleFunc("/requests/", h.GetRequestsListHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/requests/{id}", h.GetRequestByIDHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/requests/{id}/repeat", h.RepeatRequestHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/requests/{id}/scan", h.ScanRequestHandler).Methods(http.MethodPost, http.MethodOptions)

	APIPort := ":8000"

	log.Println("WebAPI is running on port " + APIPort)
	err := http.ListenAndServe(APIPort, r)
	if err != nil {
		log.Println("WebAPI failed to listen: ", err)
		return
	}

	return
}
