package routers

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	rest_api "github.com/burp_junior/internal/rest/api"
	rest_proxy "github.com/burp_junior/internal/rest/proxy"
	"github.com/burp_junior/usecase/proxy"
	"github.com/gorilla/mux"
)

func handleConnect(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handling CONNECT method")

	// Establish a TCP connection to the target server
	destConn, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	// Send a 200 OK response to the client
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	// Relay data between the client and the target server
	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func MountProxyRouter() {
	ps, err := proxy.NewProxyService()
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

func MountAPIRouter() {
	r := mux.NewRouter()

	r.HandleFunc("/requests", rest_api.APIHandler)

	APIPort := ":8000"

	log.Println("WebAPI is running on port " + APIPort)
	err := http.ListenAndServe(APIPort, r)
	if err != nil {
		log.Println("WebAPI failed to listen: ", err)
	}
}
