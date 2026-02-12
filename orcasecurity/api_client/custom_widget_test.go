package api_client

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

const errMsgExpectedNoError = "expected no error, got %v"

func TestAPIClientListCustomWidgets(t *testing.T) {
	userPrefsResp := `{
		"data": {
			"organization_preferences": [
				{"id": "org-widget-1", "name": "Org Widget 1"},
				{"id": "org-widget-2", "name": "Org Widget 2"}
			],
			"user_preferences": [
				{"id": "user-widget-1", "name": "User Widget 1"}
			]
		}
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/user_preferences" || req.URL.RawQuery != "view_type=customs_widgets" {
			t.Errorf("expected path /api/user_preferences?view_type=customs_widgets, got %s?%s", req.URL.Path, req.URL.RawQuery)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(userPrefsResp)),
		}
	})}

	apiClient := APIClient{APIEndpoint: testAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	widgets, err := apiClient.ListCustomWidgets()
	if err != nil {
		t.Fatalf(errMsgExpectedNoError, err)
	}

	if len(widgets) != 3 {
		t.Errorf("expected 3 widgets, got %d", len(widgets))
	}

	expect := []struct {
		id   string
		name string
	}{
		{"org-widget-1", "Org Widget 1"},
		{"org-widget-2", "Org Widget 2"},
		{"user-widget-1", "User Widget 1"},
	}
	for i, w := range expect {
		if widgets[i].ID != w.id || widgets[i].Name != w.name {
			t.Errorf("widget[%d]: expected id=%q name=%q, got id=%q name=%q", i, w.id, w.name, widgets[i].ID, widgets[i].Name)
		}
	}
}

func TestAPIClientListCustomWidgetsDeduplicates(t *testing.T) {
	userPrefsResp := `{
		"data": {
			"organization_preferences": [
				{"id": "dup-id", "name": "Org"}
			],
			"user_preferences": [
				{"id": "dup-id", "name": "User"}
			]
		}
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(userPrefsResp)),
		}
	})}

	apiClient := APIClient{APIEndpoint: testAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	widgets, err := apiClient.ListCustomWidgets()
	if err != nil {
		t.Fatalf(errMsgExpectedNoError, err)
	}

	if len(widgets) != 1 {
		t.Errorf("expected 1 widget (deduplicated), got %d", len(widgets))
	}
	if widgets[0].ID != "dup-id" {
		t.Errorf("expected id dup-id, got %s", widgets[0].ID)
	}
	// First occurrence (org) wins
	if widgets[0].Name != "Org" {
		t.Errorf("expected name Org (first occurrence), got %s", widgets[0].Name)
	}
}

func TestAPIClientListCustomWidgetsEmpty(t *testing.T) {
	userPrefsResp := `{
		"data": {
			"organization_preferences": [],
			"user_preferences": []
		}
	}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(userPrefsResp)),
		}
	})}

	apiClient := APIClient{APIEndpoint: testAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	widgets, err := apiClient.ListCustomWidgets()
	if err != nil {
		t.Fatalf(errMsgExpectedNoError, err)
	}
	if len(widgets) != 0 {
		t.Errorf("expected 0 widgets, got %d", len(widgets))
	}
}
