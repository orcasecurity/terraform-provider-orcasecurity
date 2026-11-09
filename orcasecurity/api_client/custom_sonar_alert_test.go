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
		c := sonarAlertStubClient(func(req *http.Request) (int, string) {
			if req.URL.Path == "/api/sonar/rules/custom" {
				return http.StatusOK, `{"status":"success","data":[],"limit":1000,"total_items":0}`
			}
			return code, `{"error":"Internal error"}`
		})
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

func TestCreateCustomAlerts_RemediationFailureDeletesRule(t *testing.T) {
	for _, deleteStatus := range []int{http.StatusOK, http.StatusInternalServerError} {
		var calls []string
		c := sonarAlertStubClient(func(req *http.Request) (int, string) {
			calls = append(calls, req.Method+" "+req.URL.Path)
			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/api/sonar/rules":
				return http.StatusOK, `{"data":{"rule_id":"r1","rule_type":"t1"}}`
			case req.Method == http.MethodPut:
				return http.StatusNotFound, `{}`
			case req.Method == http.MethodPost:
				return http.StatusBadRequest, `{"error":"bad text"}`
			default:
				return deleteStatus, `{}`
			}
		})
		create := map[string]func() error{
			"sonar": func() error {
				_, err := c.CreateCustomSonarAlert(api_client.CustomAlert{RemediationText: &api_client.CustomSonarAlertRemediationText{Text: "x"}})
				return err
			},
			"discovery": func() error {
				_, err := c.CreateCustomDiscoveryAlert(api_client.CustomDiscoveryAlert{RemediationText: &api_client.CustomDiscoveryAlertRemediationText{Text: "x"}})
				return err
			},
		}
		for name, fn := range create {
			calls = nil
			err := fn()
			wantMsg := "the new rule r1 was deleted"
			if deleteStatus != http.StatusOK {
				wantMsg = "deleting the new rule r1 also failed"
			}
			if err == nil || !strings.Contains(err.Error(), wantMsg) {
				t.Errorf("%s delete=%d: err = %v, want %q", name, deleteStatus, err, wantMsg)
			}
			if last := calls[len(calls)-1]; !strings.HasPrefix(last, "DELETE /api/sonar/rules/r1") {
				t.Errorf("%s delete=%d: last call = %s, want the rule delete", name, deleteStatus, last)
			}
		}
	}
}
