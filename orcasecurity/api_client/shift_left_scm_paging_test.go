package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestGetAllScmPages_CachesUntilInvalidate(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data":        []map[string]string{{"id": "a"}},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	path := "/api/shiftleft/github/installations/"

	first, err := getAllScmPages[scmInstallationID](client, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || first[0].ID != "a" {
		t.Fatalf("unexpected first: %+v", first)
	}
	second, err := getAllScmPages[scmInstallationID](client, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 {
		t.Fatalf("unexpected second: %+v", second)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected cache hit (1 HTTP call), got %d", hits.Load())
	}

	client.invalidateScmListCache()
	if _, err := getAllScmPages[scmInstallationID](client, path); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("expected refetch after invalidate (2 HTTP calls), got %d", hits.Load())
	}
}

func TestGetAllScmPages_FollowsPagesUntilTotal(t *testing.T) {
	const total = 450 // > 2 pages at limit=200
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		start, _ := strconv.Atoi(r.URL.Query().Get("start_at_index"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		data := make([]map[string]string, 0, limit)
		for i := start; i < start+limit && i < total; i++ {
			data = append(data, map[string]string{"id": strconv.Itoa(i)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": total, "data": data})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != total {
		t.Fatalf("expected %d items across pages, got %d", total, len(all))
	}
	if all[0].ID != "0" || all[total-1].ID != strconv.Itoa(total-1) {
		t.Fatalf("pages stitched out of order: first=%s last=%s", all[0].ID, all[total-1].ID)
	}
	if hits.Load() != 3 {
		t.Fatalf("expected 3 page fetches (200+200+50), got %d", hits.Load())
	}
}

// An empty data page must terminate the loop even when total_items overstates
// the count, otherwise getAllScmPages would spin forever.
func TestGetAllScmPages_EmptyPageTerminates(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 9999, "data": []map[string]string{}})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty result, got %d", len(all))
	}
	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 fetch before empty-page termination, got %d", hits.Load())
	}
}
