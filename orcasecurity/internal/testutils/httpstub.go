// Package testutils holds shared unit-test scaffolding. api_client's own
// RoundTripFunc lives in that package's test binary and is not importable, so
// resource/data-source packages stubbing HTTP responses use this instead of
// copy-pasting the adapter.
package testutils

import (
	"net/http"

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

// FirstPageOnly returns body for the first page of a getAllScmPages-style
// paginated request and an empty data envelope for every page after it.
// That pagination always follows up past a non-empty page to confirm it's
// over, so any stub serving a fixed paginated envelope needs this or it
// spins to the max-page guard on the confirmation request.
func FirstPageOnly(r *http.Request, body string) string {
	if start := r.URL.Query().Get("start_at_index"); start != "" && start != "0" {
		return `{"data":[]}`
	}
	return body
}
