package api_client

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetDSPMDetector(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/scan_configuration/dspm_detector/det-1")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"det-1","organization":"org-1","title":"My Detector","details":"desc","category":"PII","sub_category":"Personal","is_disabled":false,"is_custom":true,"properties":{"conditions":[{"source":"content","operator":"match","value":"[0-9]{9}"}],"detection_types":["text","db"],"sensitivity":"high","significance":"major","keywords":["ssn"],"text_threshold":3}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	detector, err := client.GetDSPMDetector("det-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detector == nil || detector.ID != "det-1" {
		t.Fatalf("expected detector det-1, got %+v", detector)
	}
	assertDetectorFields(t, detector)
}

// assertDetectorFields checks the decoded detector returned by GetDSPMDetector.
func assertDetectorFields(t *testing.T, detector *DSPMDetector) {
	t.Helper()
	if detector.OrganizationID != "org-1" || detector.Title != "My Detector" || detector.Category != "PII" {
		t.Errorf("unexpected fields: %+v", detector)
	}
	if detector.IsDisabled || !detector.IsCustom {
		t.Errorf("unexpected flags: %+v", detector)
	}
	if len(detector.Properties.Conditions) != 1 || detector.Properties.Conditions[0].Value != "[0-9]{9}" {
		t.Errorf("unexpected conditions: %+v", detector.Properties.Conditions)
	}
	if len(detector.Properties.DetectionTypes) != 2 || detector.Properties.DetectionTypes[0] != "text" {
		t.Errorf("unexpected detection_types: %+v", detector.Properties.DetectionTypes)
	}
	if detector.Properties.TextThreshold == nil || *detector.Properties.TextThreshold != 3 {
		t.Errorf("expected text_threshold 3, got %+v", detector.Properties.TextThreshold)
	}
}

func TestGetDSPMDetector_NotFound(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"status":"failure","errors":{"id":["not found"]}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	detector, err := client.GetDSPMDetector("missing")
	if err != nil {
		t.Fatalf("expected nil error on 404 so the resource can RemoveResource, got: %v", err)
	}
	if detector != nil {
		t.Errorf("expected nil detector on 404, got %+v", detector)
	}
}

func TestCreateDSPMDetector(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "POST", "/api/scan_configuration/dspm_detector")
		body, _ := io.ReadAll(req.Body)
		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid request body: %v", err)
		}
		if payload["title"] != "My Detector" {
			t.Errorf("expected title in payload, got %v", payload["title"])
		}
		if payload["is_custom"] != true {
			t.Errorf("expected is_custom=true in payload, got %v", payload["is_custom"])
		}
		if payload["is_disabled"] != false {
			t.Errorf("expected is_disabled=false in payload, got %v", payload["is_disabled"])
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"det-1","organization":"org-1","title":"My Detector","details":"desc","category":"PII","sub_category":"Personal","is_disabled":false,"is_custom":true,"properties":{"conditions":[{"source":"content","operator":"match","value":"[0-9]{9}"}],"detection_types":["text","db"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	detector, err := client.CreateDSPMDetector(DSPMDetector{
		Title:       "My Detector",
		Details:     "desc",
		Category:    "PII",
		SubCategory: "Personal",
		IsCustom:    true,
		Properties: DSPMDetectorProperties{
			Conditions: []DSPMDetectorCondition{{Source: "content", Operator: "match", Value: "[0-9]{9}"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detector.ID != "det-1" {
		t.Errorf("expected det-1, got %s", detector.ID)
	}
	if len(detector.Properties.DetectionTypes) != 2 {
		t.Errorf("expected server default detection_types echoed back, got %+v", detector.Properties.DetectionTypes)
	}
}

func TestUpdateDSPMDetector(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "PUT", "/api/scan_configuration/dspm_detector/det-1")
		assertUpdateDetectorOmitsUnsetProperties(t, req)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":{"id":"det-1","organization":"org-1","title":"Renamed","details":"desc","category":"PII","sub_category":"Personal","is_disabled":true,"is_custom":true,"properties":{"conditions":[{"source":"content","operator":"match","value":"[0-9]{9}"}],"detection_types":["text"]}}}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	detector, err := client.UpdateDSPMDetector("det-1", DSPMDetector{Title: "Renamed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detector.Title != "Renamed" || !detector.IsDisabled {
		t.Errorf("unexpected detector: %+v", detector)
	}
}

// assertUpdateDetectorOmitsUnsetProperties pins the detector PUT contract:
// the endpoint is full-replacement, so an optional property omitted from the
// payload is CLEARED server-side. Nulled optional attributes must therefore
// be absent from the JSON (omitempty), not sent as null/zero.
func assertUpdateDetectorOmitsUnsetProperties(t *testing.T, req *http.Request) {
	t.Helper()
	body, _ := io.ReadAll(req.Body)
	var payload struct {
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid request body: %v", err)
	}
	for _, key := range []string{"keywords", "exclude_keywords", "stop_wildcards", "sensitivity", "significance", "text_threshold", "db_threshold", "ocr_threshold", "ai_threshold", "detection_types"} {
		if _, present := payload.Properties[key]; present {
			t.Errorf("%s must be omitted when unset (PUT is full-replacement; omitted means cleared): %s", key, string(body))
		}
	}
}

func TestDeleteDSPMDetector(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "DELETE", "/api/scan_configuration/dspm_detector/det-1")
		return &http.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader(``)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	if err := client.DeleteDSPMDetector("det-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDSPMDetectors(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		assertMethodPath(t, req, "GET", "/api/scan_configuration/dspm_detector")
		query := req.URL.Query()
		if query.Get("title") != "My Detector" {
			t.Errorf("expected title query param, got %q", query.Get("title"))
		}
		if query.Get("category") != "PII" {
			t.Errorf("expected category query param, got %q", query.Get("category"))
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"status":"success","data":[{"id":"det-1","title":"My Detector","category":"PII","sub_category":"Personal","is_disabled":false,"is_custom":true},{"id":"AUS_TAX_NUMBER","title":"Australian Tax Number","category":"PII","sub_category":"Government","is_disabled":false,"is_custom":false}],"total_items":2}`)),
		}
	})}

	client := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	detectors, err := client.ListDSPMDetectors(DSPMDetectorListFilters{Title: "My Detector", Category: "PII"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(detectors) != 2 {
		t.Fatalf("expected 2 detectors, got %d", len(detectors))
	}
	if detectors[1].ID != "AUS_TAX_NUMBER" || detectors[1].IsCustom {
		t.Errorf("unexpected second detector: %+v", detectors[1])
	}
}
