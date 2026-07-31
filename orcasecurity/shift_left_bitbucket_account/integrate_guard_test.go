package shift_left_bitbucket_account

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
)

// The Bitbucket integrate endpoint rejects SELECTED_REPOSITORIES without an
// explicit repositories list, which this resource never sends. The guard must
// fail before the wire with an actionable message instead of surfacing the
// backend's raw 400.
func TestIntegrateGuard_RejectsSelectedRepositories(t *testing.T) {
	ops := newOps(nil).(shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel])
	err := ops.Integrate(&resourceModel{}, api_client.ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
	})
	if err == nil {
		t.Fatal("expected the integrate guard to reject SELECTED_REPOSITORIES")
	}
	if !strings.Contains(err.Error(), "SCAN_ALL_INCLUDE_FUTURE") {
		t.Fatalf("guard error must point at the working mode, got: %v", err)
	}
}

func TestIntegrateGuard_AllowsScanAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	}))
	defer srv.Close()

	token := "stub-token"
	client, err := api_client.NewAPIClient(&srv.URL, &token)
	if err != nil {
		t.Fatal(err)
	}
	ops := newOps(client).(shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel])
	if err := ops.Integrate(&resourceModel{}, api_client.ScmInstallationUpdate{
		InstallationMode: "SCAN_ALL_INCLUDE_FUTURE",
	}); err != nil {
		t.Fatalf("SCAN_ALL_INCLUDE_FUTURE must pass the guard and reach the API: %v", err)
	}
}
