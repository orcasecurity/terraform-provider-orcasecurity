package api_client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// remediationTextServer mimics the API: PUT 404s until a row exists; POST creates it (unique per
// alert type) and answers with postStatus, committing anyway when postCommits is set.
func remediationTextServer(postStatus []int, postCommits bool, putLag int) (*APIClient, *[]string) {
	var calls []string
	exists := false
	posts := 0
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		calls = append(calls, req.Method)
		code := http.StatusOK
		switch req.Method {
		case http.MethodPut:
			if !exists || putLag > 0 {
				code = http.StatusNotFound
			}
			if exists && putLag > 0 {
				putLag--
			}
		case http.MethodPost:
			switch {
			case exists:
				code = http.StatusBadRequest
			case posts < len(postStatus):
				code = postStatus[posts]
				exists = postCommits
			default:
				exists = true
			}
			posts++
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: req}
	})}
	return &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}, &calls
}

func TestSetCustomRemediationText(t *testing.T) {
	cases := []struct {
		name        string
		postStatus  []int
		postCommits bool
		wantCalls   string
		putLag      int
		wantErr     bool
	}{
		{"new text is created", nil, false, "PUT POST", 0, false},
		{"ambiguous POST that committed is confirmed by PUT", []int{http.StatusGatewayTimeout}, true, "PUT POST PUT", 0, false},
		{"ambiguous POST that did not commit is posted again", []int{http.StatusGatewayTimeout}, false, "PUT POST PUT POST", 0, false},
		{"ambiguous POST still saving: 400 on the follow-up POST goes round again", []int{http.StatusGatewayTimeout}, true, "PUT POST PUT POST PUT", 1, false},
		{"definite rejection is not retried", []int{http.StatusBadRequest}, false, "PUT POST", 0, true},
		{"repeated ambiguity gives up", []int{http.StatusGatewayTimeout, http.StatusGatewayTimeout, http.StatusGatewayTimeout}, false, "PUT POST PUT POST PUT POST", 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stubRetrySleep(t)
			client, calls := remediationTextServer(c.postStatus, c.postCommits, c.putLag)
			err := client.SetCustomRemediationText(CustomDiscoveryAlertRemediationText{})
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if c.wantErr && len(c.postStatus) == 3 && !errors.Is(err, ErrAmbiguousWrite) {
				t.Errorf("err = %v, want ErrAmbiguousWrite", err)
			}
			if got := strings.Join(*calls, " "); got != c.wantCalls {
				t.Errorf("calls = %q, want %q", got, c.wantCalls)
			}
		})
	}
}
