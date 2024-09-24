package rest_proxy

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/burp_junior/domain"
)

var okHeader = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

type ProxyService interface {
	ParseHTTPRequest(ctx context.Context, r *http.Request) (pr *domain.HTTPRequest, err error)
	SendHTTPRequest(ctx context.Context, pr *domain.HTTPRequest) (resp *http.Response, err error)
	GetTLSConfig(ctx context.Context, pr *domain.HTTPRequest) (cfg *tls.Config, sconn *tls.Conn, err error)
}

type ProxyHandler struct {
	proxyService ProxyService
}

func NewProxyHandler(proxyService ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println("err parsing form data: ", err)
		return
	}

	pr, err := h.proxyService.ParseHTTPRequest(r.Context(), r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
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

	resp, err := h.proxyService.SendHTTPRequest(r.Context(), pr)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	err = h.ServeHTTPResponse(w, resp)
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func (h *ProxyHandler) ServeHTTPResponse(w http.ResponseWriter, resp *http.Response) (err error) {
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
		return
	}

	return
}

func (h *ProxyHandler) serveConnect(w http.ResponseWriter, r *http.Request, pr *domain.HTTPRequest) (err error) {
	tlsConf, sconn, err := h.proxyService.GetTLSConfig(r.Context(), pr)
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
