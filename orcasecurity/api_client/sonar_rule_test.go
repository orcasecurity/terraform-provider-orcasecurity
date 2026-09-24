package api_client

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const customRulesPath = "/api/sonar/rules/custom"

func customRulesPage(total int, ids ...string) string {
	rules := make([]string, len(ids))
	for i, id := range ids {
		rules[i] = fmt.Sprintf(`{"rule_id":%q}`, id)
	}
	return fmt.Sprintf(`{"status":"success","data":[%s],"limit":%d,"total_items":%d}`, strings.Join(rules, ","), customRulesPageSize, total)
}

type sonarRuleStub struct {
	ruleCodes []int
	listed    []string
	gets      int
	lists     int
}

func (s *sonarRuleStub) handle(req *http.Request) (int, string) {
	if req.URL.Path == customRulesPath {
		s.lists++
		return http.StatusOK, customRulesPage(len(s.listed), s.listed...)
	}
	code := s.ruleCodes[s.gets]
	s.gets++
	if code == http.StatusOK {
		return code, `{"data":{"rule_id":"r1","rule_type":"t1"}}`
	}
	return code, `<h1>Server Error (500)</h1>`
}

func TestGetSonarRule_ConfirmsMissingAgainstList(t *testing.T) {
	tests := []struct {
		name        string
		ruleCodes   []int
		listed      []string
		wantMissing bool
		wantErr     bool
		wantGets    int
		wantLists   int
		wantSleeps  int
	}{
		{"200 needs no confirmation", []int{200}, nil, false, false, 1, 0, 0},
		{"transient 500 then 200 is found", []int{500, 200}, nil, false, false, 2, 0, 1},
		{"500 twice and absent from list is missing", []int{500, 500}, []string{"other"}, true, false, 2, 1, 1},
		{"500 twice but still listed is an error", []int{500, 500}, []string{"r1"}, false, true, 2, 1, 1},
		{"400 and absent from list is missing", []int{400}, nil, true, false, 1, 1, 0},
		{"400 but still listed is an error", []int{400}, []string{"r1"}, false, true, 1, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slept := stubRetrySleep(t)
			stub := &sonarRuleStub{ruleCodes: tt.ruleCodes, listed: tt.listed}
			_, missing, err := pathStubClient(stub.handle).getSonarRule("r1")
			got := fmt.Sprintf("err=%v missing=%v gets=%d lists=%d sleeps=%d", err != nil, missing, stub.gets, stub.lists, len(*slept))
			want := fmt.Sprintf("err=%v missing=%v gets=%d lists=%d sleeps=%d", tt.wantErr, tt.wantMissing, tt.wantGets, tt.wantLists, tt.wantSleeps)
			if got != want {
				t.Fatalf("got %s, want %s (err: %v)", got, want, err)
			}
		})
	}
}

func TestGetSonarRule_ListFailureIsError(t *testing.T) {
	stubRetrySleep(t)
	c := pathStubClient(func(req *http.Request) (int, string) {
		if req.URL.Path == customRulesPath {
			return http.StatusInternalServerError, `<h1>Server Error (500)</h1>`
		}
		return http.StatusBadRequest, `{"error":"Internal error"}`
	})
	_, missing, err := c.getSonarRule("r1")
	if err == nil || missing {
		t.Fatalf("expected error and not missing, got missing=%v err=%v", missing, err)
	}
}

func TestCustomSonarRuleExists_PagesUntilTotal(t *testing.T) {
	var offsets []string
	c := pathStubClient(func(req *http.Request) (int, string) {
		q := req.URL.Query()
		if q.Get("limit") != fmt.Sprint(customRulesPageSize) {
			t.Errorf("limit = %q", q.Get("limit"))
		}
		offsets = append(offsets, q.Get("start_at_index"))
		first := make([]string, customRulesPageSize)
		for i := range first {
			first[i] = fmt.Sprintf("r%d", i)
		}
		if q.Get("start_at_index") == "0" {
			return http.StatusOK, customRulesPage(customRulesPageSize+2, first...)
		}
		return http.StatusOK, customRulesPage(customRulesPageSize+2, "late1", "late2")
	})

	found, err := c.customSonarRuleExists("late2")
	if err != nil || !found {
		t.Fatalf("late2 on second page: found=%v err=%v", found, err)
	}
	if strings.Join(offsets, ",") != fmt.Sprintf("0,%d", customRulesPageSize) {
		t.Fatalf("offsets = %v", offsets)
	}

	offsets = nil
	found, err = c.customSonarRuleExists("nope")
	if err != nil || found {
		t.Fatalf("nope: found=%v err=%v", found, err)
	}
	if len(offsets) != 2 {
		t.Fatalf("expected both pages scanned, got offsets %v", offsets)
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

func TestGetCustomSonarAlert_Persistent500ButListedIsError(t *testing.T) {
	stubRetrySleep(t)
	c := pathStubClient(func(req *http.Request) (int, string) {
		if req.URL.Path == customRulesPath {
			return http.StatusOK, customRulesPage(1, "r1")
		}
		return http.StatusInternalServerError, `<h1>Server Error (500)</h1>`
	})
	alert, err := c.GetCustomSonarAlert("r1")
	if err == nil || alert != nil {
		t.Fatalf("expected error for a listed rule, got alert=%v err=%v", alert, err)
	}
}
