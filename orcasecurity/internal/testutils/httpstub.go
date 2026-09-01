// Package testutils holds shared unit-test scaffolding. api_client's own
// RoundTripFunc lives in that package's test binary and is not importable, so
// resource/data-source packages stubbing HTTP responses use this instead of
// copy-pasting the adapter.
package testutils

import (
	"io"
	"net/http"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// RoundTripFunc adapts a function into an http.RoundTripper for stubbing API
// responses in tests.
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewStubAPIClient returns an APIClient whose every request is answered by fn.
func NewStubAPIClient(fn RoundTripFunc) *api_client.APIClient {
	return &api_client.APIClient{
		APIEndpoint: "http://localhost",
		APIToken:    "secret",
		HTTPClient:  &http.Client{Transport: fn},
	}
}

// JSONResponse is a one-shot http.Response for RoundTripFunc stubs.
func JSONResponse(req *http.Request, code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
