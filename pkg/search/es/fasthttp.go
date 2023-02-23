package es

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
)

// Transport implements the elastictransport interface with
// the github.com/valyala/fasthttp HTTP client.
type Transport struct{}

// RoundTrip performs the request and returns a response or error
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	freq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(freq)

	fres := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(fres)

	t.copyRequest(freq, req)

	err := fasthttp.Do(freq, fres)
	if err != nil {
		return nil, err
	}

	res := &http.Response{Header: make(http.Header)}
	t.copyResponse(res, fres)

	return res, nil
}

// copyRequest converts a http.Request to fasthttp.Request
func (t *Transport) copyRequest(dst *fasthttp.Request, src *http.Request) *fasthttp.Request {
	if src.Method == "GET" && src.Body != nil {
		src.Method = "POST"
	}

	dst.SetHost(src.Host)
	dst.SetRequestURI(src.URL.String())

	dst.Header.SetRequestURI(src.URL.String())
	dst.Header.SetMethod(src.Method)

	for k, vv := range src.Header {
		for _, v := range vv {
			dst.Header.Set(k, v)
		}
	}

	if src.Body != nil {
		dst.SetBodyStream(src.Body, -1)
	}

	return dst
}

// copyResponse converts a http.Response to fasthttp.Response
func (t *Transport) copyResponse(dst *http.Response, src *fasthttp.Response) *http.Response {
	dst.StatusCode = src.StatusCode()

	src.Header.VisitAll(func(k, v []byte) {
		dst.Header.Set(string(k), string(v))
	})

	// Cast to a string to make a copy seeing as src.Body() won't
	// be valid after the response is released back to the pool (fasthttp.ReleaseResponse).
	dst.Body = io.NopCloser(strings.NewReader(string(src.Body())))

	return dst
}

// LoggingTransport wraps our Transport to enable request logging.
type LoggingTransport struct {
	t             *Transport
	EnableLogging bool
	LogCount      int
}

func NewLoggingTransport() *LoggingTransport {
	return &LoggingTransport{t: &Transport{}, EnableLogging: true}
}

// RoundTrip executes a request, returning a response, and prints information about the flow.
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.EnableLogging {
		// Print information about the request
		fmt.Printf("> %s %s\n", req.Method, req.URL.String())
		t.LogCount++
	}

	if t.LogCount >= 2 {
		t.EnableLogging = false
	}

	return t.t.RoundTrip(req)
}
