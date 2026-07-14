package api_client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func getWithBody(t *testing.T, body string) *APIResponse {
	t.Helper()
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	})}
	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, err := client.Get("/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return resp
}

func TestReadData(t *testing.T) {
	type payload struct {
		ID string `json:"id"`
	}

	t.Run("decodes the data value", func(t *testing.T) {
		resp := getWithBody(t, `{"status":"success","data":{"id":"x-1"}}`)
		value, err := readData[payload](resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value.ID != "x-1" {
			t.Errorf("expected x-1, got %+v", value)
		}
	})

	t.Run("missing data key is an error, not a zero value", func(t *testing.T) {
		resp := getWithBody(t, `{"status":"success"}`)
		if _, err := readData[payload](resp); err == nil {
			t.Fatal("expected error on missing data key")
		}
	})

	t.Run("null data is an error, not a zero value", func(t *testing.T) {
		resp := getWithBody(t, `{"status":"success","data":null}`)
		if _, err := readData[payload](resp); err == nil {
			t.Fatal("expected error on null data")
		}
	})
}

// A well-formed envelope whose data lacks the entity id must not decode into
// a zero-value struct that Read would write into state as empty strings.
func TestGetDSPMPolicy_MissingIDIsError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{}}`)),
		}
	})}
	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if _, err := client.GetDSPMPolicy("pol-1"); err == nil {
		t.Fatal("expected error when the decoded policy has no id")
	}
}

func TestGetDSPMDetector_MissingIDIsError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{}}`)),
		}
	})}
	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if _, err := client.GetDSPMDetector("det-1"); err == nil {
		t.Fatal("expected error when the decoded detector has no id")
	}
}
