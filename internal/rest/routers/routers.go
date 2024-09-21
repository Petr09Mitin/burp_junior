package routers

import (
	"log"
	"net/http"

	rest_api "github.com/burp_junior/internal/rest/api"
	rest_proxy "github.com/burp_junior/internal/rest/proxy"
	"github.com/burp_junior/usecase/proxy"
	"github.com/burp_junior/usecase/request"
	"github.com/gorilla/mux"
)

func MountProxyRouter(rs proxy.RequestsStorage) {
	ps, err := proxy.NewProxyService(rs)
	if err != nil {
		log.Println(err)
		return
	}

	proxyHandler := rest_proxy.NewProxyHandler(ps)

	proxyPort := ":8080"
	log.Println("Proxy is running on port " + proxyPort)
	err = http.ListenAndServe(proxyPort, proxyHandler)
	if err != nil {
		log.Println("Proxy failed to listen: ", err)
	}
}

func MountAPIRouter(rs request.RequestsStorage) {
	r := mux.NewRouter()

	r.HandleFunc("/requests", rest_api.APIHandler)

	APIPort := ":8000"

	log.Println("WebAPI is running on port " + APIPort)
	err := http.ListenAndServe(APIPort, r)
	if err != nil {
		log.Println("WebAPI failed to listen: ", err)
	}
}
