package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
