package domain

import "sync"

type HTTPRequest struct {
	ID         string              `bson:"_id,omitempty"`
	Proto      string              `bson:"proto,omitempty"`
	Scheme     string              `bson:"scheme,omitempty"`
	Method     string              `bson:"method,omitempty"`
	Host       string              `bson:"host,omitempty"`
	Port       string              `bson:"port,omitempty"`
	Path       string              `bson:"path,omitempty"`
	Headers    map[string][]string `bson:"headers,omitempty"`
	GetParams  map[string][]string `bson:"get_params,omitempty"`
	PostParams map[string][]string `bson:"post_params,omitempty"`
	Cookies    map[string]string   `bson:"cookies,omitempty"`
	Body       []byte              `bson:"body,omitempty"`
}

type SafeByteArr struct {
	Buf []byte
	Mu  *sync.RWMutex
}

type SafeStringArrMap struct {
	Sam map[string][]string
	Mu  *sync.RWMutex
}

type SafeStringMap struct {
	Sm map[string]string
	Mu *sync.RWMutex
}

type SafeHTTPRequest struct {
	ID         string           `bson:"_id,omitempty"`
	Proto      string           `bson:"proto,omitempty"`
	Scheme     string           `bson:"scheme,omitempty"`
	Method     string           `bson:"method,omitempty"`
	Host       string           `bson:"host,omitempty"`
	Port       string           `bson:"port,omitempty"`
	Path       string           `bson:"path,omitempty"`
	Headers    SafeStringArrMap `bson:"headers,omitempty"`
	GetParams  SafeStringArrMap `bson:"get_params,omitempty"`
	PostParams SafeStringArrMap `bson:"post_params,omitempty"`
	Cookies    SafeStringMap    `bson:"cookies,omitempty"`
	Body       SafeByteArr      `bson:"body,omitempty"`
}

func MakeSafeHTTPRequest(req *HTTPRequest) *SafeHTTPRequest {
	return &SafeHTTPRequest{
		ID:     req.ID,
		Proto:  req.Proto,
		Scheme: req.Scheme,
		Method: req.Method,
		Host:   req.Host,
		Port:   req.Port,
		Path:   req.Path,
		Headers: SafeStringArrMap{
			Sam: req.Headers,
			Mu:  &sync.RWMutex{},
		},
		GetParams: SafeStringArrMap{
			Sam: req.GetParams,
			Mu:  &sync.RWMutex{},
		},
		PostParams: SafeStringArrMap{
			Sam: req.PostParams,
			Mu:  &sync.RWMutex{},
		},
		Cookies: SafeStringMap{
			Sm: req.Cookies,
			Mu: &sync.RWMutex{},
		},
		Body: SafeByteArr{
			Buf: req.Body,
			Mu:  &sync.RWMutex{},
		},
	}
}

func MakeHTTPRequestFromSafe(req *SafeHTTPRequest) *HTTPRequest {
	return &HTTPRequest{
		ID:         req.ID,
		Proto:      req.Proto,
		Scheme:     req.Scheme,
		Method:     req.Method,
		Host:       req.Host,
		Port:       req.Port,
		Path:       req.Path,
		Headers:    req.Headers.Sam,
		GetParams:  req.GetParams.Sam,
		PostParams: req.PostParams.Sam,
		Cookies:    req.Cookies.Sm,
		Body:       req.Body.Buf,
	}
}

func (r *HTTPRequest) GetFullHost() string {
	return r.Host + ":" + r.Port
}

type HTTPResponse struct {
	ID        string              `bson:"_id,omitempty"`
	RequestID string              `bson:"request_id,omitempty"`
	Code      int                 `bson:"code,omitempty"`
	Message   string              `bson:"message,omitempty"`
	Headers   map[string][]string `bson:"headers,omitempty"`
	Body      string              `bson:"body,omitempty"`
}
