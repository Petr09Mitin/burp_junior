package rest_proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/burp_junior/customerrors"
	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/jsonutils"
)

var okHeader = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

type RequestService interface {
	ParseHTTPRequest(ctx context.Context, r *http.Request) (pr *domain.HTTPRequest, err error)
	SendHTTPRequest(ctx context.Context, pr *domain.HTTPRequest) (resp *domain.HTTPResponse, err error)
	GetTLSConfig(ctx context.Context, pr *domain.HTTPRequest) (cfg *tls.Config, sconn *tls.Conn, err error)
	ParseHTTPResponse(ctx context.Context, resp *http.Response) (*domain.HTTPResponse, error)
	SaveRequest(ctx context.Context, r *domain.HTTPRequest) (newReq *domain.HTTPRequest, err error)
	SaveHTTPResponse(ctx context.Context, resp *domain.HTTPResponse, req *domain.HTTPRequest) (savedResp *domain.HTTPResponse, err error)
}

type SafeBuffer struct {
	buf []byte
	mu  sync.Mutex
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

	savedResp, err := h.requestService.SendHTTPRequest(r.Context(), pr)
	if err != nil {
		log.Println(err)
		jsonutils.ServeJSONError(r.Context(), w, customerrors.ErrSendingRequest)
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
	_, err = io.Copy(w, strings.NewReader(httpResponse.Body))
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

	reqBuf := &SafeBuffer{
		buf: make([]byte, 0, 100*1024),
		mu:  sync.Mutex{},
	}
	resBuf := &SafeBuffer{
		buf: make([]byte, 0, 100*1024),
		mu:  sync.Mutex{},
	}

	go transfer(sconn, cconn, wg, resBuf)
	go transfer(cconn, sconn, wg, reqBuf)

	wg.Wait()

	reqBuf.mu.Lock()
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(reqBuf.buf)))
	reqBuf.mu.Unlock()
	if err != nil {
		err = customerrors.ErrParsingRequest
		return
	}

	parsedRequest, err := h.requestService.ParseHTTPRequest(r.Context(), req)
	if err != nil {
		return
	}

	parsedRequest.Scheme = "https"
	parsedRequest.Port = "443"

	parsedRequest, err = h.requestService.SaveRequest(r.Context(), parsedRequest)
	if err != nil {
		return
	}

	resBuf.mu.Lock()
	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(resBuf.buf)), req)
	resBuf.mu.Unlock()
	if err != nil {
		err = customerrors.ErrParsingResponse
		return
	}

	parsedResponse, err := h.requestService.ParseHTTPResponse(r.Context(), res)
	if err != nil {
		return
	}

	_, err = h.requestService.SaveHTTPResponse(r.Context(), parsedResponse, parsedRequest)
	if err != nil {
		return
	}

	return
}

func transfer(reader io.Reader, writer io.Writer, wg *sync.WaitGroup, transfered *SafeBuffer) {
	defer wg.Done()
	buf := make([]byte, 10*1024)

	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Println("Error reading from connection:", err)
			return
		}

		err = nil

		if n > 0 {
			transfered.mu.Lock()
			transfered.buf = append(transfered.buf, buf[:n]...)
			transfered.mu.Unlock()

			_, err = writer.Write(buf[:n])
			if err != nil {
				log.Println("Error writing to connection:", err)
				return
			}
		} else {
			return
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
