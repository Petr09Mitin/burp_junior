package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/certs"
)

type RequestService struct {
	ca   *tls.Certificate
	reqS RequestsStorage
	resS ResponseStorage
}

type RequestsStorage interface {
	SaveRequest(ctx context.Context, r *domain.HTTPRequest) (insertedReq *domain.HTTPRequest, err error)
	GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error)
}

type ResponseStorage interface {
	SaveResponse(ctx context.Context, resp *domain.HTTPResponse) (savedResp *domain.HTTPResponse, err error)
}

func NewRequestService(reqS RequestsStorage, resS ResponseStorage) (p *RequestService, err error) {
	p = &RequestService{
		reqS: reqS,
		resS: resS,
	}
	p.ca, err = certs.GetCA("ca.crt", "ca.key")
	if err != nil {
		log.Println(err)
		return
	}

	return
}

func (p *RequestService) parseHTTPBody(r *http.Request) (body []byte, err error) {
	body = nil

	if r.Body == nil || r.ContentLength == 0 {
		return
	}

	// Handle the case where ContentLength is not set
	if r.ContentLength == -1 {
		body, err = io.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			err = fmt.Errorf("error reading request body: %v", err)
			return
		}

		err = nil
	} else {
		body = make([]byte, r.ContentLength)
		_, err = r.Body.Read(body)
		if err != nil && err != io.EOF {
			err = fmt.Errorf("error reading request body: %v", err)
			return
		}

		err = nil
	}

	return
}

func (p *RequestService) parseHTTPHeaders(r *http.Request) (headers map[string][]string, err error) {
	headers = make(map[string][]string)
	for key, values := range r.Header {
		if key == "Proxy-Connection" || key == "Cookie" {
			continue
		}

		headers[key] = values
	}

	return
}

func (p *RequestService) ParseHTTPRequest(ctx context.Context, r *http.Request) (hr *domain.HTTPRequest, err error) {
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

	hr.Headers, err = p.parseHTTPHeaders(r)
	if err != nil {
		log.Println("error parsing headers: ", err)
		return
	}

	hr.Body, err = p.parseHTTPBody(r)
	if err != nil {
		log.Println("error parsing body: ", err)
		return
	}

	hr.GetParams = r.URL.Query()

	hr.PostParams = r.PostForm

	hr.Cookies = make(map[string]string)

	for _, cookie := range r.Cookies() {
		if cookie.Name != "" {
			hr.Cookies[cookie.Name] = cookie.String()
		}
	}

	return
}

func (p *RequestService) SendHTTPRequest(ctx context.Context, hr *domain.HTTPRequest) (resp *http.Response, err error) {
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

	_, err = p.SaveRequest(ctx, hr)
	if err != nil {
		log.Println("error saving request: ", err)
		return
	}

	return
}

func (p *RequestService) GetTLSConfig(ctx context.Context, pr *domain.HTTPRequest) (tlsCfg *tls.Config, sconn *tls.Conn, err error) {
	provisionalCert, err := p.GetTLSCert(ctx, pr.Host)
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
		return p.GetTLSCert(ctx, hello.ServerName)
	}

	return
}

func (p *RequestService) GetTLSCert(ctx context.Context, host string) (cert *tls.Certificate, err error) {
	cert, err = certs.SignTLSCert(host, p.ca)
	if err != nil {
		return
	}

	return
}

func (p *RequestService) SaveRequest(ctx context.Context, r *domain.HTTPRequest) (newReq *domain.HTTPRequest, err error) {
	newReq, err = p.reqS.SaveRequest(ctx, r)
	if err != nil {
		log.Println("error saving request: ", err)
		return
	}

	return
}

func (p *RequestService) GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error) {
	reqs, err = p.reqS.GetRequestsList(ctx)
	if err != nil {
		log.Println("error getting requests list: ", err)
		return
	}

	return
}

func (r *RequestService) ParseHTTPResponse(ctx context.Context, resp *http.Response) (*domain.HTTPResponse, error) {
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Create the HTTPResponse struct
	httpResponse := &domain.HTTPResponse{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Headers: make(map[string][]string),
		Body:    body,
	}

	// Copy headers
	for key, values := range resp.Header {
		httpResponse.Headers[key] = values
	}

	return httpResponse, nil
}

func (r *RequestService) SaveHTTPResponse(ctx context.Context, resp *domain.HTTPResponse) (savedResp *domain.HTTPResponse, err error) {
	savedResp, err = r.resS.SaveResponse(ctx, resp)
	if err != nil {
		return
	}

	return
}
