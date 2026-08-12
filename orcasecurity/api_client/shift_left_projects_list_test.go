package api_client

import (
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestShiftLeftProjectSummary_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/projects/?limit=1&start_at_index=0
	// (redacted); this test exercises unmarshal shape only, not pagination.
	fixture := `{"total_items":1,"data":[{"id":"3e8339f8-7a8e-4cc2-a713-940bc2662935","name":"allscan","key":"allscan","policies":[{"id":"019ad8c7-4db3-7a53-a509-485efb9283da","name":"RK-OSS-Licensing","disabled":false,"type":"licenses","builtin":false}],"builtin":false}]}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		body := fixture
		if req.URL.Query().Get("start_at_index") != "0" {
			body = `{"total_items":1,"data":[]}`
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	projects, err := apiClient.ListShiftLeftProjects()
	if err != nil {
		t.Fatalf("ListShiftLeftProjects failed: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	p := projects[0]
	if p.ID != "3e8339f8-7a8e-4cc2-a713-940bc2662935" {
		t.Errorf("bad id: %s", p.ID)
	}
	if p.Name != "allscan" {
		t.Errorf("bad name: %s", p.Name)
	}
	if p.Key != "allscan" {
		t.Errorf("bad key: %s", p.Key)
	}
}

func TestListShiftLeftProjects_PagesUsingStartAtIndex(t *testing.T) {
	// The /api/shiftleft/projects/ endpoint ignores `offset` (confirmed live
	// against the real API) but honors `start_at_index` (the automation_v2
	// convention). Assert the loop pages with start_at_index equal to items
	// already fetched, not `offset`.
	page := func(ids ...string) string {
		items := make([]string, 0, len(ids))
		for _, id := range ids {
			items = append(items, `{"id":"`+id+`","name":"n-`+id+`","key":"k-`+id+`"}`)
		}
		return `{"total_items": 3, "data": [` + strings.Join(items, ",") + `]}`
	}

	var requestedStartAtIndex []string
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		if req.URL.Path != "/api/shiftleft/projects/" {
			t.Errorf("expected path /api/shiftleft/projects/, got %s", req.URL.Path)
		}
		if q := req.URL.Query(); q.Has("offset") {
			t.Errorf("request must not use offset (ignored by this endpoint), got %v", q)
		}
		start := req.URL.Query().Get("start_at_index")
		requestedStartAtIndex = append(requestedStartAtIndex, start)
		var body string
		switch start {
		case "0":
			body = page("p1", "p2")
		case "2":
			body = page("p3")
		default:
			body = page() // trailing empty page terminates the loop
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	projects, err := apiClient.ListShiftLeftProjects()
	if err != nil {
		t.Fatalf("ListShiftLeftProjects failed: %v", err)
	}
	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d: %+v", len(projects), projects)
	}
	if projects[0].ID != "p1" || projects[2].ID != "p3" {
		t.Errorf("unexpected project order: %+v", projects)
	}
	// A trailing empty-page fetch at start_at_index 3 confirms termination no
	// longer trusts total_items.
	want := []string{"0", "2", "3"}
	if !slices.Equal(requestedStartAtIndex, want) {
		t.Errorf("expected start_at_index %v, got %v", want, requestedStartAtIndex)
	}
}

func TestListShiftLeftProjects_StopsOnEmptyPage(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"total_items": 50, "data": []}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	projects, err := apiClient.ListShiftLeftProjects()
	if err != nil {
		t.Fatalf("ListShiftLeftProjects failed: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}
