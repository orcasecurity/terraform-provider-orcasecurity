package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Concurrent lookups on a cached slice must not race when stamping installation_id.
func TestFindScmUnit_ConcurrentNoRace(t *testing.T) {
	const instID = "inst-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data":        []map[string]string{{"id": "acc-1", "account_id": "target-slug", "account_name": "n"}},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	if _, err := client.FindBitbucketAccountBySlug(instID, "no-such-slug"); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acc, err := client.FindBitbucketAccountBySlug(instID, "target-slug")
			if err != nil || acc == nil || acc.InstallationID != instID {
				t.Errorf("bad result: acc=%+v err=%v", acc, err)
			}
		}()
	}
	wg.Wait()
}

// Cold cache + singleflight: stamp must use a per-caller copy.
func TestFindScmUnit_ConcurrentNoRace_ColdCache(t *testing.T) {
	const instID = "inst-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data":        []map[string]string{{"id": "acc-1", "account_id": "target-slug", "account_name": "n"}},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acc, err := client.FindBitbucketAccountBySlug(instID, "target-slug")
			if err != nil || acc == nil || acc.InstallationID != instID {
				t.Errorf("bad result: acc=%+v err=%v", acc, err)
			}
		}()
	}
	wg.Wait()
}

// Same as above via listScmUnitsByInstallation.
func TestListBitbucketAccounts_ConcurrentNoRace_ColdCache(t *testing.T) {
	const instID = "inst-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		if strings.Contains(r.URL.Path, "integrated_accounts") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total_items": 1,
				"data":        []map[string]string{{"id": "acc-1", "account_id": "target-slug", "account_name": "n"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_items": 1,
			"data":        []map[string]string{{"id": instID}},
		})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			accounts, err := client.ListBitbucketAccounts()
			if err != nil || len(accounts) != 1 || accounts[0].InstallationID != instID {
				t.Errorf("bad result: accounts=%+v err=%v", accounts, err)
			}
		}()
	}
	wg.Wait()
}

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

// Concurrent calls for the same path must collapse into a single HTTP fetch.
func TestGetAllScmPages_SingleFlightCollapsesStampede(t *testing.T) {
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

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := getAllScmPages[scmInstallationID](client, path); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if hits.Load() != 1 {
		t.Fatalf("expected exactly 1 HTTP call across 16 concurrent fetches, got %d", hits.Load())
	}
}

// A response omitting total_items must not terminate after the first full page
// (absent must not read as 0); it paginates until an empty page.
func TestGetAllScmPages_AbsentTotalItems(t *testing.T) {
	const total = 250 // one full page (200) + a short page (50)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, _ := strconv.Atoi(r.URL.Query().Get("start_at_index"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		data := make([]map[string]string, 0, limit)
		for i := start; i < start+limit && i < total; i++ {
			data = append(data, map[string]string{"id": strconv.Itoa(i)})
		}
		// Deliberately omit total_items.
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != total {
		t.Fatalf("expected %d items with absent total_items, got %d", total, len(all))
	}
}

// A server that always returns a full page (bogus/inflated total) must abort at
// the max-page guard instead of paging forever.
func TestGetAllScmPages_MaxPageGuard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]map[string]string, 0, 200)
		for i := 0; i < 200; i++ {
			data = append(data, map[string]string{"id": strconv.Itoa(i)})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 999999999, "data": data})
	}))
	defer srv.Close()

	client := &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}
	_, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/")
	if err == nil || !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected max-page guard error, got %v", err)
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
