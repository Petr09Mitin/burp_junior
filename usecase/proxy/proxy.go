package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/burp_junior/domain"
)

type ProxyService struct {
	ca *tls.Certificate
	rs RequestsStorage
}

type RequestsStorage interface{}

func NewProxyService(rs RequestsStorage) (p *ProxyService, err error) {
	p = &ProxyService{
		rs: rs,
	}
	p.ca, err = GetCA("ca.crt", "ca.key")
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func (p *ProxyService) ParseHTTPRequest(r *http.Request) (hr *domain.HTTPRequest, err error) {
	hr = &domain.HTTPRequest{}
	hr.Method = r.Method

	if colonIdx := strings.Index(r.Host, ":"); colonIdx == -1 {
		hr.Host = r.Host
		hr.Port = "80"
	} else {
		hr.Host = r.Host[:colonIdx]
		hr.Port = r.Host[colonIdx+1:]
	}

	hr.Scheme = r.URL.Scheme
	hr.Proto = r.Proto

	// Parse path
	hr.Path = r.URL.Path

	hr.Headers = make(map[string][]string)
	for key, values := range r.Header {
		if key == "Proxy-Connection" {
			continue
		}

		hr.Headers[key] = values
	}

	hr.Body = nil

	if r.Body == nil || r.ContentLength == 0 {
		return
	}

	// Handle the case where ContentLength is not set
	if r.ContentLength == -1 {
		hr.Body, err = io.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			err = fmt.Errorf("error reading request body: %v", err)
			return
		}

		err = nil
	} else {
		hr.Body = make([]byte, r.ContentLength)
		_, err = r.Body.Read(hr.Body)
		if err != nil && err != io.EOF {
			err = fmt.Errorf("error reading request body: %v", err)
			return
		}

		err = nil
	}

	return
}

func (p *ProxyService) SendHTTPRequest(hr *domain.HTTPRequest) (resp *http.Response, err error) {
	client := &http.Client{}
	req, err := http.NewRequest(hr.Method, "", bytes.NewReader(hr.Body))
	if err != nil {
		err = fmt.Errorf("Error creating request: %v\n", err)
		return
	}

	req.URL.Host = hr.GetFullHost()
	req.URL.Path = hr.Path
	req.URL.Scheme = hr.Scheme

	req.Proto = hr.Proto

	for key, values := range hr.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err = client.Do(req)
	if err != nil {
		err = fmt.Errorf("Error sending request: %v\n", err)
		return
	}

	return
}

func (p *ProxyService) GetTLSConfig(pr *domain.HTTPRequest) (tlsCfg *tls.Config, sconn *tls.Conn, err error) {
	provisionalCert, err := p.GetTLSCert(pr.Host)
	if err != nil {
		return
	}

	tlsCfg = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	tlsCfg.Certificates = []tls.Certificate{*provisionalCert}

	tlsCfg.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		cConfig.ServerName = hello.ServerName
		sconn, err = tls.Dial("tcp", pr.GetFullHost(), cConfig)
		if err != nil {
			return nil, err
		}
		return p.GetTLSCert(hello.ServerName)
	}

	return
}

func (p *ProxyService) GetTLSCert(host string) (cert *tls.Certificate, err error) {
	cert, err = SignTLSCert(host, p.ca)
	if err != nil {
		return
	}

	return
}
