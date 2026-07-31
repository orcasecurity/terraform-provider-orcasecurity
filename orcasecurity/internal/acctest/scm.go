package acctest

import (
	"os"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RestoreScmBody uses Adopt so restores follow project_id XOR policies.
func RestoreScmBody(common api_client.ScmUnitCommonFields) api_client.ScmInstallationUpdate {
	return shift_left_integration.Adopt(
		types.StringNull(),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		shift_left_integration.ProjectIntent{},
		common,
	)
}

// APIClient reads ORCASECURITY_API_* for live-state snapshot/restore tests.
//
// It is the single entry point for tests that mutate a real tenant directly, rather than
// through resource.Test, so it also carries the TF_ACC gate those tests would otherwise
// get for free. Without it, `go test ./...` on a machine that happens to have credentials
// exported would write to a live tenant.
func APIClient(t *testing.T) *api_client.APIClient {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set; skipping test that reads and writes a live Orca tenant")
	}
	endpoint := strings.TrimRight(os.Getenv("ORCASECURITY_API_ENDPOINT"), "/")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	if endpoint == "" || token == "" {
		t.Skip("ORCASECURITY_API_ENDPOINT / ORCASECURITY_API_TOKEN not set")
	}
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to create API client for acceptance test: %s", err)
	}
	return client
}
