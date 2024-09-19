package routers

import (
	"log"
	"net/http"

	rest_api "github.com/burp_junior/internal/rest/api"
	rest_proxy "github.com/burp_junior/internal/rest/proxy"
	"github.com/burp_junior/usecase/proxy"
	"github.com/gorilla/mux"
)

func MountProxyRouter() {
	r := mux.NewRouter()

	proxyHandler := rest_proxy.NewProxyHandler(proxy.NewProxyService())

	r.HandleFunc("/", proxyHandler.HandleProxy)

	proxyPort := ":8080"
	log.Println("Proxy is running on port " + proxyPort)
	err := http.ListenAndServe(proxyPort, r)
	if err != nil {
		log.Fatalln("Proxy failed to listen: ", err)
	}
}

func MountAPIRouter() {
	r := mux.NewRouter()

	r.HandleFunc("/requests", rest_api.APIHandler)

	APIPort := ":8000"

	log.Println("Proxy is running on port " + APIPort)
	err := http.ListenAndServe(APIPort, r)
	if err != nil {
		log.Fatalln("WebAPI failed to listen: ", err)
	}
}
