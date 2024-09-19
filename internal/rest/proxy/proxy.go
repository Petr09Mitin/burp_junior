package proxy

import (
	"net/http"
)

func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}
