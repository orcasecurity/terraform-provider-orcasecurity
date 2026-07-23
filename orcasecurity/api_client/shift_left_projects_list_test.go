package api_client

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestShiftLeftProjectSummary_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/projects/?limit=1&start_at_index=0
	// (redacted), with total_items trimmed to 1 so the single page is complete
	// and this test exercises unmarshal shape only, not pagination.
	fixture := `{"total_items":1,"data":[{"id":"3e8339f8-7a8e-4cc2-a713-940bc2662935","name":"allscan","key":"allscan","policies":[{"id":"019ad8c7-4db3-7a53-a509-485efb9283da","name":"RK-OSS-Licensing","disabled":false,"type":"licenses","builtin":false}],"builtin":false}]}`

	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(fixture)),
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
		body := page("p1", "p2")
		if start != "0" {
			body = page("p3")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(body)),
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
	if len(requestedStartAtIndex) != 2 || requestedStartAtIndex[0] != "0" || requestedStartAtIndex[1] != "2" {
		t.Errorf("expected start_at_index [0 2], got %v", requestedStartAtIndex)
	}
}

func TestListShiftLeftProjects_StopsOnEmptyPage(t *testing.T) {
	// Defensive: if the server claims more total_items than it returns, an
	// empty page must terminate the loop rather than spin forever.
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{"total_items": 50, "data": []}`)),
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
