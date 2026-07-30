package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// automationPage renders one page of the /api/automations envelope.
func automationPage(total int, ids ...string) map[string]any {
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{
			"id":     id,
			"name":   "auto-" + id,
			"status": "enabled",
			"filter": map[string]any{
				"sonar_query": map[string]any{"models": []string{"Alert"}, "type": "object_set"},
			},
			"actions": []any{},
		})
	}
	return map[string]any{"total_items": total, "data": data}
}

// Adding server-side filters changed how paginateOffset builds its query, from a
// fmt.Sprintf to url.Values.Encode — which sorts keys and percent-escapes. Every
// caller that passes nil filters (ListAutomationsV2 and all the SCM lists) must
// keep the byte-identical query it sent before filters existed. Assertions via
// URL.Query().Get() cannot catch an extra, renamed, or re-ordered key, so pin
// RawQuery instead.
func TestPaginateOffset_NilFiltersSendOnlyPagingParams(t *testing.T) {
	var rawQueries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/automations" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		rawQueries = append(rawQueries, r.URL.RawQuery)
		// Three rows over two short pages, so the second request's
		// start_at_index is pinned as well as the first's.
		if len(rawQueries) == 1 {
			_ = json.NewEncoder(w).Encode(automationPage(3, "a1", "a2"))
			return
		}
		_ = json.NewEncoder(w).Encode(automationPage(3, "a3"))
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	automations, err := client.ListAutomationsV2()
	if err != nil {
		t.Fatalf("ListAutomationsV2: %v", err)
	}
	if len(automations) != 3 {
		t.Fatalf("expected 3 automations, got %d", len(automations))
	}

	want := []string{"limit=300&start_at_index=0", "limit=300&start_at_index=2"}
	if len(rawQueries) != len(want) {
		t.Fatalf("expected %d requests, got %v", len(want), rawQueries)
	}
	for i, q := range want {
		if rawQueries[i] != q {
			t.Errorf("request %d query = %q, want %q", i, rawQueries[i], q)
		}
	}
}

// The server caps limit at its own maximum (300 for OrcaLimitOffsetPaginator), so a caller
// asking for more gets short pages. start_at_index must therefore advance by rows actually
// received: advancing by the requested limit would skip everything the server declined to
// serve, and treating a short page as the last page would truncate the list.
func TestPaginateOffset_ShortPagesFromClampedLimitDoNotTruncate(t *testing.T) {
	const serverMaxLimit = 3
	var offsets []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offsets = append(offsets, r.URL.Query().Get("start_at_index"))
		rows := []map[string]string{}
		for i := 0; i < serverMaxLimit; i++ {
			id := len(offsets)*serverMaxLimit + i - serverMaxLimit
			if id >= 7 {
				break
			}
			rows = append(rows, map[string]string{"id": fmt.Sprintf("row-%d", id)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 7, "data": rows})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	// Ask for 10 per page against a server that will only ever serve 3.
	rows, err := paginateOffset[scmInstallationID](client, "/api/things", nil, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 7 {
		t.Fatalf("expected all 7 rows across clamped pages, got %d: %+v", len(rows), rows)
	}
	for i, row := range rows {
		if want := fmt.Sprintf("row-%d", i); row.ID != want {
			t.Fatalf("row %d = %q, want %q (duplicated or skipped rows)", i, row.ID, want)
		}
	}
	if want := []string{"0", "3", "6"}; !slices.Equal(offsets, want) {
		t.Errorf("start_at_index sequence = %v, want %v", offsets, want)
	}
}

// A filtered call must add its filters alongside — never instead of — the paging
// params, so a filtered lookup still walks every page of the narrowed set.
func TestPaginateOffset_FiltersKeepPagingParams(t *testing.T) {
	var rawQueries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQueries = append(rawQueries, r.URL.RawQuery)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data":        []map[string]string{{"id": "row-1"}},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	if _, err := paginateOffset[scmInstallationID](
		client, "/api/things", listFilters{"gitlab_project_id": "7"}, 200, 10); err != nil {
		t.Fatal(err)
	}

	// Encode sorts by key: gitlab_project_id, limit, start_at_index.
	const want = "gitlab_project_id=7&limit=200&start_at_index=0"
	if len(rawQueries) != 1 || rawQueries[0] != want {
		t.Errorf("queries = %v, want [%q]", rawQueries, want)
	}
}
