package api_client_test

import (
	"net/http"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
)

func TestListExternalServiceConfigIDs_IgnoresRowsOfOtherServices(t *testing.T) {
	client := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return testutils.JSONResponse(req, http.StatusOK, `{"status":"success","data":[
			{"id":"other","service_name":"slack","template_name":"t","config":"not an object"},
			{"id":"mine","service_name":"opsgenie","template_name":" t ","config":{}}]}`)
	})
	ids, err := api_client.ListExternalServiceConfigIDs(client, "opsgenie", "t")
	if err != nil || len(ids) != 1 || !ids["mine"] {
		t.Fatalf("ids=%v err=%v, want only the matching row", ids, err)
	}
}
