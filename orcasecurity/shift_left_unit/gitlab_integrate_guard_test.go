package shift_left_unit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
)

// The GitLab integrate endpoint accepts only ALWAYS/NEVER for skip_check_runs;
// ONLY_ON_INTERNAL_ISSUE is update-only. The guard must fail before the wire
// with an actionable message instead of surfacing the backend's raw 400.
func TestIntegrateGuard_RejectsOnlyOnInternalIssue(t *testing.T) {
	ops := gitlabOps(nil).(shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, gitlabGroupModel])
	err := ops.IntegrateGuard(api_client.ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		ConfigSettings:   api_client.ShiftLeftConfigSettings{SkipCheckRuns: "ONLY_ON_INTERNAL_ISSUE"},
	})
	if err == nil {
		t.Fatal("expected the integrate guard to reject ONLY_ON_INTERNAL_ISSUE")
	}
	if !strings.Contains(err.Error(), "ALWAYS or NEVER") {
		t.Fatalf("guard error must name the accepted values, got: %v", err)
	}
}

func TestIntegrateGuard_AllowsAlways(t *testing.T) {
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
	ops := gitlabOps(client).(shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, gitlabGroupModel])
	body := api_client.ScmInstallationUpdate{
		InstallationMode: "SELECTED_REPOSITORIES",
		ConfigSettings:   api_client.ShiftLeftConfigSettings{SkipCheckRuns: "ALWAYS"},
	}
	if err := ops.IntegrateGuard(body); err != nil {
		t.Fatalf("ALWAYS must pass the guard: %v", err)
	}
	if err := ops.Integrate(&gitlabGroupModel{}, body); err != nil {
		t.Fatalf("ALWAYS must reach the API: %v", err)
	}
}
