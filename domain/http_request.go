package domain

type HTTPRequest struct {
	Proto   string
	Scheme  string
	Method  string
	Host    string
	Port    string
	Path    string
	Headers map[string][]string
	Body    []byte
}
