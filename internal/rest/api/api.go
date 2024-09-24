package rest_api

import (
	"context"
	"log"
	"net/http"

	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/jsonutils"
)

type APIHandler struct {
	rs RequestService
}

type RequestService interface {
	GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error)
}

func NewAPIHandler(rs RequestService) *APIHandler {
	return &APIHandler{
		rs: rs,
	}
}

func (h *APIHandler) GetRequestsListHandler(w http.ResponseWriter, r *http.Request) {
	rl, err := h.rs.GetRequestsList(r.Context())
	if err != nil {
		log.Println("error getting requests list: ", err)
		return
	}

	jsonutils.ServeJSONBody(r.Context(), w, rl, http.StatusOK)

	return
}

func (h *APIHandler) GetSingleRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

func (h *APIHandler) RepeatRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}

func (h *APIHandler) ScanRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	return
}
