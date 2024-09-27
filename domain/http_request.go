package domain

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
