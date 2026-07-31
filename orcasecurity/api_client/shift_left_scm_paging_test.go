package api_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// The global SCM unit list serializers omit installation_id, so the client stamps
// it from the installation it fetched under. Both lookup paths must stamp, or the
// installation_id a caller needs for for_each comes back empty.
func TestScmUnitLookups_StampInstallationID(t *testing.T) {
	const instID = "inst-1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("findScmUnitBy", func(t *testing.T) {
		acc, err := client.FindBitbucketAccountBySlug(instID, "target-slug")
		if err != nil {
			t.Fatal(err)
		}
		if acc == nil {
			t.Fatal("account not found")
		}
		if acc.InstallationID != instID {
			t.Errorf("installation_id = %q, want %q", acc.InstallationID, instID)
		}
	})

	t.Run("listScmUnitsByInstallation", func(t *testing.T) {
		accounts, err := client.ListBitbucketAccounts()
		if err != nil {
			t.Fatal(err)
		}
		if len(accounts) != 1 {
			t.Fatalf("got %d accounts, want 1", len(accounts))
		}
		if accounts[0].InstallationID != instID {
			t.Errorf("installation_id = %q, want %q", accounts[0].InstallationID, instID)
		}
	})
}

func newBitbucketAccountsSearchServer(t *testing.T, accountsPath string, searchHonored bool) (*APIClient, *[]string) {
	t.Helper()
	var searches []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != accountsPath {
			_ = json.NewEncoder(w).Encode(map[string]any{"total_items": 0, "data": []map[string]string{}})
			return
		}
		search := r.URL.Query().Get("search")
		searches = append(searches, search+"|"+r.URL.Query().Get("search_fields"))
		rows := []map[string]string{{"id": "acc-1", "account_id": "target-slug", "account_name": "Target"}}
		if search != "" && !searchHonored {
			// Stands in for a search that cannot see the field the caller matches on.
			rows = []map[string]string{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_items": len(rows), "data": rows})
	}))
	t.Cleanup(srv.Close)
	return &APIClient{APIEndpoint: srv.URL, HTTPClient: srv.Client()}, &searches
}

// A unit lookup by name asks the server to narrow the list first, so the common case does not walk
// every page of accounts. The search covers one name field per SCM, so a filtered miss must fall back
// to the whole list rather than report the unit as absent.
func TestFindScmUnitByName_FiltersServerSideThenFallsBack(t *testing.T) {
	const instID = "inst-1"
	accountsPath := "/api/shiftleft/bitbucket/installations/" + instID + "/integrated_accounts/"

	t.Run("narrows the list with a name search", func(t *testing.T) {
		client, searches := newBitbucketAccountsSearchServer(t, accountsPath, true)
		acc, err := client.FindBitbucketAccountBySlug(instID, "target-slug")
		if err != nil {
			t.Fatalf("expected the account, got %+v (%v)", acc, err)
		}
		if acc == nil {
			t.Fatalf("expected the account, got %+v (%v)", acc, err)
		}
		if len(*searches) != 1 {
			t.Fatalf("expected a single name-filtered request, got %v", *searches)
		}
		if (*searches)[0] != "target-slug|name" {
			t.Fatalf("expected a single name-filtered request, got %v", *searches)
		}
	})

	t.Run("falls back to the unfiltered list on a filtered miss", func(t *testing.T) {
		client, searches := newBitbucketAccountsSearchServer(t, accountsPath, false)
		acc, err := client.FindBitbucketAccountBySlug(instID, "target-slug")
		if err != nil {
			t.Fatal(err)
		}
		if acc == nil {
			t.Fatal("a filtered miss must not be reported as an absent account")
		}
		if len(*searches) != 2 {
			t.Fatalf("expected a filtered attempt then an unfiltered scan, got %v", *searches)
		}
		if (*searches)[1] != "|" {
			t.Fatalf("expected a filtered attempt then an unfiltered scan, got %v", *searches)
		}
	})
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
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/", nil)
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
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/", nil)
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
	_, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/", nil)
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
	all, err := getAllScmPages[scmInstallationID](client, "/api/shiftleft/github/installations/", nil)
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
