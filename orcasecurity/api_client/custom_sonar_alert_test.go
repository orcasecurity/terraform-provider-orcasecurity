package api_client_test

import (
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func sonarAlertStubClient(handler func(req *http.Request) (int, string)) *api_client.APIClient {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		code, body := handler(req)
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}
	return &api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
}

func TestGetCustomSonarAlert_MissingReturnsNil(t *testing.T) {
	for _, code := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		c := sonarAlertStubClient(func(*http.Request) (int, string) { return code, `{"error":"Internal error"}` })
		alert, err := c.GetCustomSonarAlert("1")
		if err != nil {
			t.Fatalf("status %d: unexpected error %v", code, err)
		}
		if alert != nil {
			t.Fatalf("status %d: expected nil alert, got %+v", code, alert)
		}
	}
}

func TestGetCustomSonarAlert_ReadsRuleAndRemediation(t *testing.T) {
	var calls []string
	c := sonarAlertStubClient(func(req *http.Request) (int, string) {
		calls = append(calls, req.Method+" "+req.URL.Path)
		if strings.HasPrefix(req.URL.Path, "/api/sonar/rules/") {
			return http.StatusOK, `{"data":{"rule_id":"1","name":"n","rule":"AwsS3Bucket","rule_type":"t1","enabled":true}}`
		}
		return http.StatusOK, `{"alert_type":"t1","enabled":true,"custom_text":"fix"}`
	})
	alert, err := c.GetCustomSonarAlert("1")
	if err != nil {
		t.Fatal(err)
	}
	if alert.ID != "1" || alert.Rule != "AwsS3Bucket" || alert.RemediationText.Text != "fix" {
		t.Fatalf("unexpected alert %+v", alert)
	}
	want := "GET /api/sonar/rules/1,GET /api/alerts/custom_remediation_text"
	if got := strings.Join(calls, ","); got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
}
