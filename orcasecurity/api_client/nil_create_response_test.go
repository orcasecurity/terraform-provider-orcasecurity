package api_client_test

import (
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
)

func TestCreateCustomTagRule_ReadBack404IsAnError(t *testing.T) {
	client := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return testutils.JSONResponse(req, 200, `{"data":{"tags_rule_id":"rule-1"},"status":"success"}`)
		}
		return testutils.JSONResponse(req, 404, `{"error":"not found"}`)
	})
	rule, err := client.CustomTagRuleCreateOp(api_client.CustomTagRule{Name: "r", RuleType: api_client.CustomTagRuleRuleTypeString}).Create()
	if rule != nil || err == nil || !strings.Contains(err.Error(), "created as rule-1") || !strings.Contains(err.Error(), "import it") {
		t.Fatalf("rule=%v err=%v, want a created-as-rule-1 import error", rule, err)
	}
}

func TestCreateCustomTagRule_ReadBackFailureNamesTheID(t *testing.T) {
	client := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return testutils.JSONResponse(req, 200, `{"data":{"tags_rule_id":"rule-1"},"status":"success"}`)
		}
		return testutils.JSONResponse(req, 403, `{"error":"forbidden"}`)
	})
	rule, err := client.CustomTagRuleCreateOp(api_client.CustomTagRule{Name: "r", RuleType: api_client.CustomTagRuleRuleTypeString}).Create()
	if rule != nil || err == nil || !strings.Contains(err.Error(), "created as rule-1") || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("rule=%v err=%v, want a created-as-rule-1 error carrying the GET failure", rule, err)
	}
}

func TestCreateCustomTagRule_MissingIDIsAnErrorWithoutReadBack(t *testing.T) {
	gets := 0
	client := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return testutils.JSONResponse(req, 200, `{"data":{},"status":"success"}`)
		}
		gets++
		return testutils.JSONResponse(req, 404, `{}`)
	})
	rule, err := client.CustomTagRuleCreateOp(api_client.CustomTagRule{Name: "r", RuleType: api_client.CustomTagRuleRuleTypeString}).Create()
	if rule != nil || err == nil || !strings.Contains(err.Error(), "missing tags_rule_id") {
		t.Fatalf("rule=%v err=%v, want a missing-tags_rule_id error", rule, err)
	}
	if gets != 0 {
		t.Errorf("made %d read-back GETs, want none", gets)
	}
}

func TestCreateShiftLeftCveExceptionList_NonOKIsAnError(t *testing.T) {
	client := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return testutils.JSONResponse(req, 400, `{"name":["already exists"]}`)
	})
	list, err := client.ShiftLeftCveExceptionListCreateOp(api_client.ShiftLeftCveExceptionList{Name: "l"}).Create()
	if list != nil || err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("list=%v err=%v, want a 400 error", list, err)
	}
}
