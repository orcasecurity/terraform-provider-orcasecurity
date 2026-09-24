package api_client

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func pathStubClient(handler func(req *http.Request) (int, string)) *APIClient {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		code, body := handler(req)
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}
	})}
	return &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
}

func TestGetCustomDiscoveryAlert_MissingReturnsNil(t *testing.T) {
	stubRetrySleep(t)
	for _, code := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		c := pathStubClient(func(*http.Request) (int, string) { return code, `{"error":"Internal error"}` })
		alert, err := c.GetCustomDiscoveryAlert("r0000000000")
		if err != nil {
			t.Fatalf("status %d: unexpected error %v", code, err)
		}
		if alert != nil {
			t.Fatalf("status %d: expected nil alert, got %+v", code, alert)
		}
	}
}

func TestGetCustomDiscoveryAlert_ThrottledIsError(t *testing.T) {
	stubRetrySleep(t)
	c := pathStubClient(func(*http.Request) (int, string) {
		return http.StatusTooManyRequests, `{"status":"failure","error_code":"throttled","message":"Request was throttled. Expected available in 1 second.","errors":{}}`
	})
	alert, err := c.GetCustomDiscoveryAlert("r1")
	if err == nil || alert != nil {
		t.Fatalf("expected throttle error and nil alert, got alert=%v err=%v", alert, err)
	}
}

func TestGetCustomDiscoveryAlert_ReadsRuleAndRemediation(t *testing.T) {
	var paths []string
	c := pathStubClient(func(req *http.Request) (int, string) {
		paths = append(paths, req.Method+" "+req.URL.Path)
		if strings.HasPrefix(req.URL.Path, "/api/sonar/rules/") {
			return http.StatusOK, `{"data":{"rule_id":"r1","name":"n","rule_type":"t1"}}`
		}
		return http.StatusOK, `{"alert_type":"t1","enabled":true,"custom_text":"fix"}`
	})
	alert, err := c.GetCustomDiscoveryAlert("r1")
	if err != nil {
		t.Fatal(err)
	}
	if alert.ID != "r1" || alert.RemediationText.Text != "fix" {
		t.Fatalf("unexpected alert %+v", alert)
	}
	want := []string{"GET /api/sonar/rules/r1", "GET /api/alerts/custom_remediation_text"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("calls = %v, want %v", paths, want)
	}
}

func TestGetAlertCategories_CachedAcrossCallsAndClones(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	c := pathStubClient(func(*http.Request) (int, string) {
		mu.Lock()
		calls++
		mu.Unlock()
		return http.StatusOK, `{"data":["Network misconfigurations","IAM misconfigurations"]}`
	})
	c.alertCategories = &alertCategoryCache{}
	clone := c.withHTTPTimeout(c.HTTPClient.Timeout)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(client *APIClient) {
			defer wg.Done()
			got, err := client.GetAlertCategories()
			if err != nil || len(got) != 2 {
				t.Errorf("got %v, %v", got, err)
			}
		}([]*APIClient{c, clone}[i%2])
	}
	wg.Wait()
	if calls != 1 {
		t.Fatalf("category endpoint calls = %d, want 1", calls)
	}
}

func TestGetAlertCategories_ErrorNotCached(t *testing.T) {
	calls := 0
	c := pathStubClient(func(*http.Request) (int, string) {
		calls++
		if calls == 1 {
			return http.StatusBadRequest, `{"error":"no"}`
		}
		return http.StatusOK, `{"data":["a"]}`
	})
	c.alertCategories = &alertCategoryCache{}
	if _, err := c.GetAlertCategories(); err == nil {
		t.Fatal("expected first call to fail")
	}
	got, err := c.GetAlertCategories()
	if err != nil || len(got) != 1 {
		t.Fatalf("second call: %v, %v", got, err)
	}
}
