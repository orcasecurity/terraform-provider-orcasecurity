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
	retryMaxDelay       = 16 * time.Second
	retryAfterCap        = 2 * time.Minute
)

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

// roundTripWithRetry performs the HTTP round trip with retries for transient
// transport failures and selected HTTP status codes (408, 429, 502, 503, 504).
// On success (any HTTP status), returns a fully read APIResponse; err is only
// for request body read failures, transport failures after retries, or context
// cancellation during backoff.
func (c *APIClient) roundTripWithRetry(req http.Request) (*APIResponse, error) {
	ctx := req.Context()

	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}

	for attempt := 0; attempt < maxHTTPRetryAttempts; attempt++ {
		r := req.Clone(ctx)
		if reqBody != nil {
			b := reqBody
			r.Body = io.NopCloser(bytes.NewReader(b))
			r.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(b)), nil
			}
			r.ContentLength = int64(len(b))
		}

		res, execErr := c.Execute(*r)
		if execErr != nil {
			if attempt < maxHTTPRetryAttempts-1 && isRetriableRoundTripError(execErr) {
				if err := sleepCtx(ctx, retryDelay(attempt, nil)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, execErr
		}

		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			if attempt < maxHTTPRetryAttempts-1 && isRetriableRoundTripError(readErr) {
				if err := sleepCtx(ctx, retryDelay(attempt, nil)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, readErr
		}

		apiResp := &APIResponse{_body: body, response: res}

		if apiResp.IsOk() {
			return apiResp, nil
		}
		if !isRetriableHTTPStatus(res.StatusCode) {
			return apiResp, nil
		}
		if attempt == maxHTTPRetryAttempts-1 {
			return apiResp, nil
		}
		if err := sleepCtx(ctx, retryDelay(attempt, res)); err != nil {
			return nil, err
		}
	}

	return nil, errors.New("orca api client: retry loop exhausted")
}
