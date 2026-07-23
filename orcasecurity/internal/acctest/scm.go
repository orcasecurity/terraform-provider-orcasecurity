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
func RestoreScmBody(mode string, defaultPolicies bool, policies []api_client.ScmPolicyRef, project *api_client.ScmProjectRef, cfg api_client.ShiftLeftConfigSettings) api_client.ScmInstallationUpdate {
	return shift_left_integration.Adopt(
		types.StringNull(),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		shift_left_integration.ProjectIntent{},
		shift_left_integration.ExistingFromAPI(mode, defaultPolicies, policies, project, cfg),
	).Body
}

// APIClient reads ORCASECURITY_API_* for live-state snapshot/restore tests.
func APIClient(t *testing.T) *api_client.APIClient {
	t.Helper()
	endpoint := strings.TrimRight(os.Getenv("ORCASECURITY_API_ENDPOINT"), "/")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to create API client for acceptance test: %s", err)
	}
	return client
}
