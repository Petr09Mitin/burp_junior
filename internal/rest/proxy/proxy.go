package rest_proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/burp_junior/customerrors"
	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/jsonutils"
)

var okHeader = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

type RequestService interface {
	ParseHTTPRequest(ctx context.Context, r *http.Request) (pr *domain.HTTPRequest, err error)
	SendHTTPRequest(ctx context.Context, pr *domain.HTTPRequest) (resp *http.Response, err error)
	GetTLSConfig(ctx context.Context, pr *domain.HTTPRequest) (cfg *tls.Config, sconn *tls.Conn, err error)
	ParseHTTPResponse(ctx context.Context, resp *http.Response) (*domain.HTTPResponse, error)
	SaveHTTPResponse(ctx context.Context, resp *domain.HTTPResponse, req *domain.HTTPRequest) (savedResp *domain.HTTPResponse, err error)
}

type ProxyHandler struct {
	requestService RequestService
}

func NewProxyHandler(requestService RequestService) *ProxyHandler {
	return &ProxyHandler{
		requestService: requestService,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("err parsing form data: ", err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrParsingFormData)
		return
	}

	pr, err := h.requestService.ParseHTTPRequest(r.Context(), r)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrParsingRequest)
		return
	}

	if pr.Method == http.MethodConnect {
		err = h.serveConnect(w, r, pr)
		if err != nil {
			log.Println("connect err:", err)
			return
		}

		return
	}

	resp, err := h.requestService.SendHTTPRequest(r.Context(), pr)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrSendingRequest)
		return
	}
	defer resp.Body.Close()

	parsedResp, err := h.requestService.ParseHTTPResponse(r.Context(), resp)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrParsingResponse)
		return
	}

	savedResp, err := h.requestService.SaveHTTPResponse(r.Context(), parsedResp, pr)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrSavingResponse)
		return
	}

	err = h.ServeHTTPResponse(w, savedResp)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrServingResponse)
		return
	}

	return
}

func (h *ProxyHandler) ServeHTTPResponse(w http.ResponseWriter, httpResponse *domain.HTTPResponse) (err error) {
	// Write status code
	w.WriteHeader(httpResponse.Code)

	// Write headers
	for key, values := range httpResponse.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write body
	_, err = io.Copy(w, bytes.NewReader(httpResponse.Body))
	if err != nil {
		return
	}

	return
}

func (h *ProxyHandler) serveConnect(w http.ResponseWriter, r *http.Request, pr *domain.HTTPRequest) (err error) {
	tlsConf, sconn, err := h.requestService.GetTLSConfig(r.Context(), pr)
	if err != nil {
		return
	}

	cconn, err := handshake(w, tlsConf)
	if err != nil {
		return
	}

	defer cconn.Close()

	if sconn == nil {
		sconn, err = tls.Dial("tcp", pr.GetFullHost(), tlsConf)
		if err != nil {
			log.Println("dial", pr.GetFullHost(), err)
			return
		}
	}
	defer sconn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go transfer(sconn, cconn, wg)
	go transfer(cconn, sconn, wg)

	wg.Wait()

	return
}

func transfer(reader io.Reader, writer io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, 100*1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from connection:", err)
			}
			return
		}
		if n > 0 {
			_, err = writer.Write(buf[:n])
			if err != nil {
				log.Println("Error writing to connection:", err)
				return
			}
		}
	}
}

func handshake(w http.ResponseWriter, config *tls.Config) (net.Conn, error) {
	raw, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "no upstream", 503)
		return nil, err
	}

	if _, err = raw.Write(okHeader); err != nil {
		raw.Close()
		return nil, err
	}

	conn := tls.Server(raw, config)
	err = conn.Handshake()
	if err != nil {
		conn.Close()
		raw.Close()
		return nil, err
	}
	return conn, nil
}
