package api_client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIsRetriableHTTPStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusRequestTimeout, true},
		{http.StatusBadRequest, false},
		{http.StatusNotFound, false},
		{http.StatusInternalServerError, false},
	}
	for _, tt := range tests {
		if got := isRetriableHTTPStatus(tt.code); got != tt.want {
			t.Errorf("isRetriableHTTPStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestRoundTripWithRetry_502Then200(t *testing.T) {
	var n int
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		n++
		if n < 3 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`<html>502</html>`)),
				Header:     make(http.Header),
				Request:    req,
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}

	c := &APIClient{
		APIEndpoint: "http://localhost",
		APIToken:    "secret",
		HTTPClient:  httpClient,
	}
	req, err := http.NewRequest(http.MethodGet, "http://localhost/api/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatalf("roundTripWithRetry: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 HTTP attempts, got %d", n)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode())
	}
}

func TestRoundTripWithRetry_NoRetryOn400(t *testing.T) {
	var n int
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		n++
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":"no"}`)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}

	c := &APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	req, _ := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 attempt, got %d", n)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode())
	}
}

func TestRoundTripWithRetry_POSTBodyPreservedAcrossRetries(t *testing.T) {
	var n int
	wantBody := `{"a":1}`
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		b, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if string(b) != wantBody {
			t.Errorf("attempt %d: body = %q", n, string(b))
		}
		n++
		if n == 1 {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`err`)),
				Header:     make(http.Header),
				Request:    req,
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}

	c := &APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/r", strings.NewReader(wantBody))
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("attempts = %d", n)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode())
	}
}
