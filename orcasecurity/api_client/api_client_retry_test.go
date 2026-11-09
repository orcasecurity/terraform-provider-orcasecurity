package api_client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	retryTestAPIEndpoint  = "http://localhost"
	errFmtRetryTestStatus = "status = %d"
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
		APIEndpoint: retryTestAPIEndpoint,
		APIToken:    "secret",
		HTTPClient:  httpClient,
	}
	req, err := http.NewRequest(http.MethodGet, retryTestAPIEndpoint+"/api/x", nil)
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
		t.Fatalf(errFmtRetryTestStatus, resp.StatusCode())
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

	c := &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	req, _ := http.NewRequest(http.MethodGet, retryTestAPIEndpoint+"/", nil)
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 attempt, got %d", n)
	}
	if resp.StatusCode() != http.StatusBadRequest {
		t.Fatalf(errFmtRetryTestStatus, resp.StatusCode())
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

	c := &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	req, _ := http.NewRequest(http.MethodPost, retryTestAPIEndpoint+"/r", strings.NewReader(wantBody))
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("attempts = %d", n)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf(errFmtRetryTestStatus, resp.StatusCode())
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

type errRoundTripFunc func(*http.Request) (*http.Response, error)

func (f errRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIsRetriableRoundTripError_TimeoutFlag(t *testing.T) {
	err := timeoutError{}
	c := &APIClient{}
	if !c.isRetriableRoundTripError(err) {
		t.Fatal("timeout should be retriable by default")
	}
	c.disableTimeoutRetry = true
	if c.isRetriableRoundTripError(err) {
		t.Fatal("timeout should not be retriable when disableTimeoutRetry is set")
	}
	if !c.isRetriableRoundTripError(io.ErrUnexpectedEOF) {
		t.Fatal("unexpected EOF should still be retriable when timeouts are not")
	}
}

func TestRoundTripWithRetry_TimeoutNotRetriedWhenDisabled(t *testing.T) {
	var n int
	httpClient := &http.Client{Transport: errRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		n++
		return nil, timeoutError{}
	})}
	c := &APIClient{
		APIEndpoint:         retryTestAPIEndpoint,
		APIToken:            "secret",
		HTTPClient:          httpClient,
		disableTimeoutRetry: true,
	}
	req, _ := http.NewRequest(http.MethodPost, retryTestAPIEndpoint+"/r", nil)
	_, err := c.roundTripWithRetry(*req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if n != 1 {
		t.Fatalf("expected 1 attempt when timeout retry is disabled, got %d", n)
	}
}

func stubRetrySleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var slept []time.Duration
	origSleep, origJitter := retrySleep, retryJitter
	retrySleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}
	retryJitter = func(time.Duration) time.Duration { return 0 }
	t.Cleanup(func() { retrySleep, retryJitter = origSleep, origJitter })
	return &slept
}

func statusSequenceClient(codes func(n int) int) (*APIClient, *int) {
	n := 0
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		n++
		code := codes(n)
		body := `{}`
		if code == http.StatusTooManyRequests {
			body = `{"status":"failure","error_code":"throttled","message":"Request was throttled. Expected available in 1 second.","errors":{}}`
		}
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}
	return &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}, &n
}

func TestRoundTripWithRetry_429RetriesBeyondDefaultBudget(t *testing.T) {
	stubRetrySleep(t)
	c, n := statusSequenceClient(func(n int) int {
		if n <= 7 {
			return http.StatusTooManyRequests
		}
		return http.StatusOK
	})
	req, _ := http.NewRequest(http.MethodGet, retryTestAPIEndpoint+"/api/sonar/rules/x", nil)
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if *n != 8 {
		t.Fatalf("attempts = %d, want 8", *n)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf(errFmtRetryTestStatus, resp.StatusCode())
	}
}

func TestRoundTripWithRetry_429GivesUpAfterThrottleBudget(t *testing.T) {
	slept := stubRetrySleep(t)
	c, n := statusSequenceClient(func(int) int { return http.StatusTooManyRequests })
	req, _ := http.NewRequest(http.MethodGet, retryTestAPIEndpoint+"/api/sonar/rules/x", nil)
	resp, err := c.roundTripWithRetry(*req)
	if err != nil {
		t.Fatal(err)
	}
	if *n != maxThrottleRetryAttempts {
		t.Fatalf("attempts = %d, want %d", *n, maxThrottleRetryAttempts)
	}
	if len(*slept) != maxThrottleRetryAttempts-1 {
		t.Fatalf("sleeps = %d, want %d", len(*slept), maxThrottleRetryAttempts-1)
	}
	if resp.StatusCode() != http.StatusTooManyRequests {
		t.Fatalf(errFmtRetryTestStatus, resp.StatusCode())
	}
}

func TestRoundTripWithRetry_502KeepsDefaultBudget(t *testing.T) {
	stubRetrySleep(t)
	c, n := statusSequenceClient(func(int) int { return http.StatusBadGateway })
	req, _ := http.NewRequest(http.MethodGet, retryTestAPIEndpoint+"/api/x", nil)
	if _, err := c.roundTripWithRetry(*req); err != nil {
		t.Fatal(err)
	}
	if *n != maxHTTPRetryAttempts {
		t.Fatalf("attempts = %d, want %d", *n, maxHTTPRetryAttempts)
	}
}

func TestRetryDelay_ThrottleWithoutRetryAfter(t *testing.T) {
	stubRetrySleep(t)
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header)}
	want := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second}
	for attempt, w := range want {
		if got := retryDelay(attempt, resp); got != w {
			t.Errorf("attempt %d: delay = %v, want %v", attempt, got, w)
		}
	}
}

func TestRetryDelay_HonorsRetryAfter(t *testing.T) {
	stubRetrySleep(t)
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"3"}}}
	if got := retryDelay(0, resp); got != 3*time.Second {
		t.Fatalf("delay = %v, want 3s", got)
	}
}

func TestRetryDelay_JitterIsBounded(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header)}
	for i := 0; i < 200; i++ {
		got := retryDelay(0, resp)
		if got < throttleBaseDelay || got >= throttleBaseDelay+throttleBaseDelay/2 {
			t.Fatalf("delay = %v, want in [1s, 1.5s)", got)
		}
	}
}
