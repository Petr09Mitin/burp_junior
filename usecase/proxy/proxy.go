package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/burp_junior/domain"
)

type ProxyService struct{}

func NewProxyService() *ProxyService {
	return &ProxyService{}
}

func (p *ProxyService) ParseHTTPRequest(r *http.Request) (hr *domain.HTTPRequest, err error) {
	hr = &domain.HTTPRequest{}
	hr.Host = r.Host
	hr.Port = "80" // Default port
	if r.TLS != nil {
		hr.Port = "443" // Default port for HTTPS
	}
	if colonIndex := len(hr.Host) - len(r.URL.Host); colonIndex > 0 {
		hr.Host = r.URL.Host[:colonIndex]
		hr.Port = r.URL.Host[colonIndex+1:]
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

	log.Println("Parsed request: ", hr)

	return
}

func (p *ProxyService) SendHTTPRequest(hr *domain.HTTPRequest) (resp *http.Response, err error) {
	client := &http.Client{}
	req, err := http.NewRequest(hr.Method, "", bytes.NewReader(hr.Body))
	if err != nil {
		err = fmt.Errorf("Error creating request: %v\n", err)
		return
	}

	req.URL.Host = hr.Host + ":" + hr.Port
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
