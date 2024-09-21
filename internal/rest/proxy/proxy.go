package rest_proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/burp_junior/domain"
)

var okHeader = []byte("HTTP/1.1 200 Connection Established\r\n\r\n")

type IProxyService interface {
	ParseHTTPRequest(r *http.Request) (pr *domain.HTTPRequest, err error)
	SendHTTPRequest(pr *domain.HTTPRequest) (resp *http.Response, err error)
	GetTLSConfig(pr *domain.HTTPRequest) (cfg *tls.Config, sconn *tls.Conn, err error)
}

type ProxyHandler struct {
	proxyService IProxyService
}

func NewProxyHandler(proxyService IProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pr, err := h.proxyService.ParseHTTPRequest(r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if pr.Method == http.MethodConnect {
		err = h.serveConnect(w, pr)
		if err != nil {
			log.Println("connect err:", err)
			return
		}

		return
	}

	resp, err := h.proxyService.SendHTTPRequest(pr)
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

func (h *ProxyHandler) serveConnect(w http.ResponseWriter, pr *domain.HTTPRequest) (err error) {
	// established connect tunnel
	w.Write([]byte(okHeader))

	tlsConf, sconn, err := h.proxyService.GetTLSConfig(pr)
	if err != nil {
		return
	}

	cconn, err := handshake(w, tlsConf)
	if err != nil {
		return
	}

	defer cconn.Close()

	if sconn == nil {
		cConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		sconn, err = tls.Dial("tcp", pr.GetFullHost(), cConfig)
		if err != nil {
			log.Println("dial", pr.GetFullHost(), err)
			return
		}
	}
	defer sconn.Close()

	done := make(chan struct{})

	go func() {
		io.Copy(cconn, sconn)

		log.Println("client --> proxy --> server")
		data := make([]byte, 1024)
		for {
			log.Println("client --> proxy")
			n, err := cconn.Read(data)
			if err != nil {
				log.Printf("cconn -> sconn write error: %v\n", err)
				done <- struct{}{}
			}
			log.Println("proxy --> server")
			sconn.Write(data[:n])
		}
	}()

	go func() {
		io.Copy(sconn, cconn)

		log.Println("server --> proxy --> client")
		data := make([]byte, 1024)
		for {
			log.Println("server --> proxy")
			for {
				n, err := sconn.Read(data)
				if err != nil {
					log.Printf("%v\n", err)
					done <- struct{}{}
				}
				log.Println("proxy --> client")
				cconn.Write(data[:n])
				if n < 1024 {
					break
				}
			}
		}
	}()

	<-done
	log.Println("Tunnel with " + pr.GetFullHost() + " closed")

	return
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func handshake(w http.ResponseWriter, config *tls.Config) (net.Conn, error) {
	raw, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "no upstream", 503)
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
