package rest_api

import (
	"context"
	"log"
	"net/http"

	"github.com/burp_junior/customerrors"
	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/jsonutils"
	"github.com/gorilla/mux"
)

type APIHandler struct {
	rs RequestService
}

type RequestService interface {
	GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error)
	GetRequestByID(ctx context.Context, reqID string) (req *domain.HTTPRequest, err error)
	RepeatRequestByID(ctx context.Context, reqID string) (res *domain.HTTPResponse, err error)
	ScanRequestWithCommandInjection(ctx context.Context, reqID string) (unsafeReq *domain.HTTPRequest, err error)
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

func (h *APIHandler) GetRequestByIDHandler(w http.ResponseWriter, r *http.Request) {
	reqID, ok := mux.Vars(r)["id"]
	if !ok {
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrInvalidRequest)
		return
	}

	req, err := h.rs.GetRequestByID(r.Context(), reqID)
	if err != nil {
		jsonutils.ServeJSONError(r.Context(), w, err)
		return
	}

	jsonutils.ServeJSONBody(r.Context(), w, req, http.StatusOK)

	return
}

func (h *APIHandler) RepeatRequestHandler(w http.ResponseWriter, r *http.Request) {
	reqID, ok := mux.Vars(r)["id"]
	if !ok {
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrInvalidRequest)
		return
	}

	res, err := h.rs.RepeatRequestByID(r.Context(), reqID)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, err)
		return
	}

	jsonutils.ServeJSONBody(r.Context(), w, res, http.StatusCreated)

	return
}

func (h *APIHandler) ScanRequestHandler(w http.ResponseWriter, r *http.Request) {
	reqID, ok := mux.Vars(r)["id"]
	if !ok {
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrInvalidRequest)
		return
	}

	unsafeReq, err := h.rs.ScanRequestWithCommandInjection(r.Context(), reqID)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, err)
		return
	}

	jsonutils.ServeJSONBody(r.Context(), w, unsafeReq, http.StatusCreated)

	return
}
