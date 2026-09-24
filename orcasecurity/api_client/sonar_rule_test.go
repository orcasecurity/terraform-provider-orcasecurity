package api_client

import (
	"net/http"
	"testing"
)

func TestGetSonarRule_Recheck500(t *testing.T) {
	tests := []struct {
		name        string
		codes       []int
		wantMissing bool
		wantCalls   int
		wantSleeps  int
	}{
		{"transient 500 then 200 is found", []int{500, 200}, false, 2, 1},
		{"500 twice is missing", []int{500, 500}, true, 2, 1},
		{"400 is missing without recheck", []int{400}, true, 1, 0},
		{"200 needs no recheck", []int{200}, false, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slept := stubRetrySleep(t)
			calls := 0
			c := pathStubClient(func(*http.Request) (int, string) {
				code := tt.codes[calls]
				calls++
				if code == http.StatusOK {
					return code, `{"data":{"rule_id":"r1","rule_type":"t1"}}`
				}
				return code, `<h1>Server Error (500)</h1>`
			})
			_, missing, err := c.getSonarRule("r1")
			if err != nil {
				t.Fatal(err)
			}
			if missing != tt.wantMissing || calls != tt.wantCalls || len(*slept) != tt.wantSleeps {
				t.Fatalf("missing=%v calls=%d sleeps=%d, want %v/%d/%d",
					missing, calls, len(*slept), tt.wantMissing, tt.wantCalls, tt.wantSleeps)
			}
		})
	}
}

func TestGetCustomDiscoveryAlert_Transient500IsNotDeleted(t *testing.T) {
	stubRetrySleep(t)
	ruleCalls := 0
	c := pathStubClient(func(req *http.Request) (int, string) {
		if req.URL.Path == "/api/sonar/rules/r1" {
			ruleCalls++
			if ruleCalls == 1 {
				return http.StatusInternalServerError, `<h1>Server Error (500)</h1>`
			}
			return http.StatusOK, `{"data":{"rule_id":"r1","rule_type":"t1"}}`
		}
		return http.StatusOK, `{"alert_type":"t1","enabled":true,"custom_text":"fix"}`
	})
	alert, err := c.GetCustomDiscoveryAlert("r1")
	if err != nil || alert == nil || alert.ID != "r1" {
		t.Fatalf("expected alert r1 after transient 500, got %+v, %v", alert, err)
	}
}

func TestGetCustomSonarAlert_Transient500IsNotDeleted(t *testing.T) {
	stubRetrySleep(t)
	ruleCalls := 0
	c := pathStubClient(func(req *http.Request) (int, string) {
		if req.URL.Path == "/api/sonar/rules/r1" {
			ruleCalls++
			if ruleCalls == 1 {
				return http.StatusInternalServerError, `<h1>Server Error (500)</h1>`
			}
			return http.StatusOK, `{"data":{"rule_id":"r1","rule_type":"t1","rule":"AzureVNet"}}`
		}
		return http.StatusOK, `{"alert_type":"t1","enabled":true,"custom_text":"fix"}`
	})
	alert, err := c.GetCustomSonarAlert("r1")
	if err != nil || alert == nil || alert.ID != "r1" {
		t.Fatalf("expected alert r1 after transient 500, got %+v, %v", alert, err)
	}
}
