package request

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/burp_junior/domain"
	"github.com/burp_junior/pkg/certs"
)

var (
	commandInjectionScans = []string{
		";cat /etc/passwd;",
		"|cat /etc/passwd|",
		"`cat /etc/passwd`",
	}
	commandInjectionCheckString = "root:"
)

type RequestService struct {
	ca   *tls.Certificate
	reqS RequestsStorage
	resS ResponseStorage
}

type SafeInjections struct {
	mu *sync.RWMutex
	ci []string
}

type RequestsStorage interface {
	SaveRequest(ctx context.Context, r *domain.HTTPRequest) (insertedReq *domain.HTTPRequest, err error)
	GetRequestsList(ctx context.Context) (reqs []*domain.HTTPRequest, err error)
	GetRequestByID(ctx context.Context, id string) (req *domain.HTTPRequest, err error)
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
		return
	}

	return
}

func (p *RequestService) parseCookie(cookieStr string) (*http.Cookie, error) {
	// Create a dummy HTTP response string with the cookie
	responseStr := "HTTP/1.1 200 OK\r\n" +
		"Set-Cookie: " + cookieStr + "\r\n" +
		"\r\n"

	// Read the response using http.ReadResponse
	reader := bufio.NewReader(strings.NewReader(responseStr))
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return nil, err
	}

	// Extract the cookie from the response
	if len(resp.Cookies()) > 0 {
		return resp.Cookies()[0], nil
	}

	return nil, errors.New("no cookies found")
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
			return
		}

		err = nil
	} else {
		body = make([]byte, r.ContentLength)
		_, err = r.Body.Read(body)
		if err != nil && err != io.EOF {
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

	err = r.ParseForm()
	if err != nil {
		log.Println("error parsing form data: ", err)
		return
	}
	hr.PostParams = r.PostForm

	hr.Method = r.Method

	if colonIdx := strings.Index(r.Host, ":"); colonIdx == -1 {
		hr.Host = r.Host
		hr.Port = "80"
	} else {
		hr.Host = r.Host[:colonIdx]
		hr.Port = r.Host[colonIdx+1:]
	}

	hr.Scheme = r.URL.Scheme
	if hr.Scheme == "" {
		hr.Scheme = "http"
		if r.TLS != nil || hr.Port == "443" {
			hr.Scheme = "https"
		}
	}

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

	hr.Cookies = make(map[string]string)

	for _, cookie := range r.Cookies() {
		if cookie.Name != "" {
			hr.Cookies[cookie.Name] = cookie.String()
		}
	}

	return
}

func (r *RequestService) SendHTTPRequest(ctx context.Context, req *domain.HTTPRequest) (res *domain.HTTPResponse, err error) {
	var tlsCfg *tls.Config

	if req.Scheme == "https" {
		tlsCfg, _, err = r.GetTLSConfig(ctx, req)
		if err != nil {
			return
		}
	}

	tr := &http.Transport{
		MaxIdleConns:    10,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: tlsCfg,
	}
	client := &http.Client{Transport: tr}

	var bodyReader io.Reader

	if len(req.PostParams) > 0 {
		form := url.Values{}

		for key, values := range req.PostParams {
			for _, value := range values {
				form.Add(key, value)
			}
		}

		bodyReader = strings.NewReader(form.Encode())
	} else {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.Scheme+"://"+req.GetFullHost()+req.Path, bodyReader)
	if err != nil {
		return
	}

	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	q := httpReq.URL.Query()
	for key, values := range req.GetParams {
		for _, value := range values {
			q.Add(key, value)
		}
	}
	httpReq.URL.RawQuery = q.Encode()

	for _, value := range req.Cookies {
		cookie, err := r.parseCookie(value)
		if err != nil {
			return nil, err
		}

		httpReq.AddCookie(cookie)
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return
	}

	res, err = r.ParseHTTPResponse(ctx, httpResp)
	if err != nil {
		return
	}

	res, err = r.SaveHTTPResponse(ctx, res, req)
	if err != nil {
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
	var bodyReader io.ReadCloser
	var err error

	// Check if the response body is gzip-encoded
	if resp.Header.Get("Content-Encoding") == "gzip" {
		bodyReader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
	} else {
		bodyReader = resp.Body
	}

	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	defer bodyReader.Close()

	// Create the HTTPResponse struct
	httpResponse := &domain.HTTPResponse{
		Code:    resp.StatusCode,
		Message: resp.Status,
		Headers: make(map[string][]string),
		Body:    string(body),
	}

	// Copy headers
	for key, values := range resp.Header {
		httpResponse.Headers[key] = values
	}

	return httpResponse, nil
}

func (r *RequestService) SaveHTTPResponse(ctx context.Context, resp *domain.HTTPResponse, req *domain.HTTPRequest) (savedResp *domain.HTTPResponse, err error) {
	resp.RequestID = req.ID

	savedResp, err = r.resS.SaveResponse(ctx, resp)
	if err != nil {
		return
	}

	return
}

func (r *RequestService) GetRequestByID(ctx context.Context, reqID string) (req *domain.HTTPRequest, err error) {
	req, err = r.reqS.GetRequestByID(ctx, reqID)
	if err != nil {
		return
	}

	return
}

func (r *RequestService) RepeatRequestByID(ctx context.Context, reqID string) (res *domain.HTTPResponse, err error) {
	req, err := r.GetRequestByID(ctx, reqID)
	if err != nil {
		return
	}

	res, err = r.SendHTTPRequest(ctx, req)
	if err != nil {
		return
	}

	return
}

func (r *RequestService) isCommandInjectionVulnerable(resp *domain.HTTPResponse) bool {
	return strings.Contains(resp.Body, commandInjectionCheckString)
}

func copySyncMapIntoStringArrMap(sm *sync.Map) (rm map[string][]string) {
	rm = make(map[string][]string, 0)
	sm.Range(func(key any, value any) bool {
		rm[key.(string)] = value.([]string)
		return true
	})

	return
}

func copySyncMapIntoStringMap(sm *sync.Map) (rm map[string]string) {
	rm = make(map[string]string, 0)
	sm.Range(func(key any, value any) bool {
		rm[key.(string)] = value.(string)
		return true
	})

	return
}

// ScanRequestWithCommandInjection scans request with ID=reqID, sequentially pasting Command Injection
// variations into every Header, Cookie, Get param and FormData field.
// It returns unsafeReq of type *domain.HTTPRequest, fields of which contain only values that are penetrated by injection.
func (r *RequestService) ScanRequestWithCommandInjection(ctx context.Context, reqID string) (unsafeReq *domain.HTTPRequest, err error) {
	req, err := r.reqS.GetRequestByID(ctx, reqID)
	if err != nil {
		return
	}

	unsafeR := *req
	ci := SafeInjections{
		mu: &sync.RWMutex{},
		ci: commandInjectionScans,
	}

	headers := &sync.Map{}
	cookies := &sync.Map{}
	getParams := &sync.Map{}
	postParams := &sync.Map{}

	safeR := domain.MakeSafeHTTPRequest(&unsafeR)

	globalWg := &sync.WaitGroup{}
	globalWg.Add(4)

	go r.scanHeadersWorker(ctx, globalWg, safeR, ci, headers)
	go r.scanCookiesWorker(ctx, globalWg, safeR, ci, cookies)
	go r.scanGetParamsWorker(ctx, globalWg, safeR, ci, getParams)
	go r.scanPostParamsWorker(ctx, globalWg, safeR, ci, postParams)

	globalWg.Wait()

	unsafeReq = &unsafeR

	unsafeReq.Headers = copySyncMapIntoStringArrMap(headers)
	unsafeReq.GetParams = copySyncMapIntoStringArrMap(getParams)
	unsafeReq.PostParams = copySyncMapIntoStringArrMap(postParams)
	unsafeReq.Cookies = copySyncMapIntoStringMap(cookies)

	return
}
