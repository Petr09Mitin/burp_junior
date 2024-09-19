package rest_proxy

import (
	"fmt"
	"io"
	"net/http"

	"github.com/burp_junior/domain"
)

type IProxyService interface {
	ParseHTTPRequest(r *http.Request) (pr *domain.HTTPRequest, err error)
	SendHTTPRequest(pr *domain.HTTPRequest) (resp *http.Response, err error)
}

type ProxyHandler struct {
	proxyService IProxyService
}

func NewProxyHandler(proxyService IProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
	}
}

func (h *ProxyHandler) HandleProxy(w http.ResponseWriter, r *http.Request) {
	pr, err := h.proxyService.ParseHTTPRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	resp, err := h.proxyService.SendHTTPRequest(pr)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy headers from the response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write the status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error copying response body", http.StatusInternalServerError)
		return
	}

	return
}
