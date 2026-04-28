package api_client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	maxHTTPRetryAttempts = 5
	retryBaseDelay       = 250 * time.Millisecond
	retryMaxDelay        = 16 * time.Second
	retryAfterCap        = 2 * time.Minute
)

func slurpRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	reqBody, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	return reqBody, nil
}

func cloneRequestWithBody(ctx context.Context, proto *http.Request, reqBody []byte) http.Request {
	r := proto.Clone(ctx)
	if reqBody == nil {
		return *r
	}
	b := reqBody
	r.Body = io.NopCloser(bytes.NewReader(b))
	r.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	r.ContentLength = int64(len(b))
	return *r
}

func readResponseBody(res *http.Response) ([]byte, error) {
	defer func() { _ = res.Body.Close() }()
	return io.ReadAll(res.Body)
}

// sleepIfRetriableTransportError backs off when err is retriable and attempts remain.
// Returns retry=true to run another attempt; retry=false with nil err to surface errOut;
// or retry=false with non-nil err on context cancellation during sleep.
func sleepIfRetriableTransportError(ctx context.Context, attempt int, errOut error) (retry bool, err error) {
	if attempt >= maxHTTPRetryAttempts-1 || !isRetriableRoundTripError(errOut) {
		return false, nil
	}
	if err := sleepCtx(ctx, retryDelay(attempt, nil)); err != nil {
		return false, err
	}
	return true, nil
}

// httpResponseFinalOrBackoff returns the API response when the caller should stop (success,
// non-retriable HTTP status, or last attempt). If retry is true, backoff was applied and
// another attempt should run.
func httpResponseFinalOrBackoff(ctx context.Context, attempt int, body []byte, res *http.Response) (apiResp *APIResponse, retry bool, err error) {
	apiResp = &APIResponse{_body: body, response: res}
	if apiResp.IsOk() {
		return apiResp, false, nil
	}
	if !isRetriableHTTPStatus(res.StatusCode) || attempt == maxHTTPRetryAttempts-1 {
		return apiResp, false, nil
	}
	if err := sleepCtx(ctx, retryDelay(attempt, res)); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

// transportPhaseOutcome maps Execute / body-read errors to either another attempt
// (continueLoop) or a terminal error for the caller to return.
func transportPhaseOutcome(ctx context.Context, attempt int, phaseErr error) (continueLoop bool, err error) {
	retry, sleepErr := sleepIfRetriableTransportError(ctx, attempt, phaseErr)
	if sleepErr != nil {
		return false, sleepErr
	}
	if retry {
		return true, nil
	}
	return false, phaseErr
}

// httpResponsePhaseOutcome maps HTTP status handling to continue, terminal error, or success.
func httpResponsePhaseOutcome(ctx context.Context, attempt int, body []byte, res *http.Response) (apiResp *APIResponse, continueLoop bool, err error) {
	apiResp, retry, sleepErr := httpResponseFinalOrBackoff(ctx, attempt, body, res)
	if sleepErr != nil {
		return nil, false, sleepErr
	}
	if retry {
		return nil, true, nil
	}
	return apiResp, false, nil
}

func isRetriableHTTPStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetriableRoundTripError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		if ne.Timeout() {
			return true
		}
		if t, ok := ne.(interface{ Temporary() bool }); ok && t.Temporary() {
			return true
		}
	}
	return false
}

func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if s, err := strconv.ParseInt(ra, 10, 64); err == nil && s > 0 {
				d := time.Duration(s) * time.Second
				if d > retryAfterCap {
					d = retryAfterCap
				}
				return d
			}
		}
	}
	shift := attempt
	if shift > 6 {
		shift = 6
	}
	d := retryBaseDelay * time.Duration(1<<uint(shift))
	if d > retryMaxDelay {
		return retryMaxDelay
	}
	return d
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// roundTripIteration runs a single HTTP attempt (execute, read body, apply HTTP retry policy).
// If tryAgain is true, the caller should run another attempt. If tryAgain is false and err is nil,
// resp is the final APIResponse. If err is non-nil, the caller must return that error.
func (c *APIClient) roundTripIteration(ctx context.Context, attempt int, proto *http.Request, reqBody []byte) (resp *APIResponse, tryAgain bool, err error) {
	r := cloneRequestWithBody(ctx, proto, reqBody)
	res, execErr := c.Execute(r)
	if execErr != nil {
		cont, out := transportPhaseOutcome(ctx, attempt, execErr)
		if cont {
			return nil, true, nil
		}
		return nil, false, out
	}

	body, readErr := readResponseBody(res)
	if readErr != nil {
		cont, out := transportPhaseOutcome(ctx, attempt, readErr)
		if cont {
			return nil, true, nil
		}
		return nil, false, out
	}

	apiResp, cont, out := httpResponsePhaseOutcome(ctx, attempt, body, res)
	if out != nil {
		return nil, false, out
	}
	if cont {
		return nil, true, nil
	}
	return apiResp, false, nil
}

// roundTripWithRetry performs the HTTP round trip with retries for transient
// transport failures and selected HTTP status codes (408, 429, 502, 503, 504).
// On success (any HTTP status), returns a fully read APIResponse; err is only
// for request body read failures, transport failures after retries, or context
// cancellation during backoff.
func (c *APIClient) roundTripWithRetry(req http.Request) (*APIResponse, error) {
	ctx := req.Context()
	reqBody, err := slurpRequestBody(&req)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < maxHTTPRetryAttempts; attempt++ {
		resp, tryAgain, iterErr := c.roundTripIteration(ctx, attempt, &req, reqBody)
		if iterErr != nil {
			return nil, iterErr
		}
		if tryAgain {
			continue
		}
		return resp, nil
	}

	return nil, errors.New("orca api client: retry loop exhausted")
}
