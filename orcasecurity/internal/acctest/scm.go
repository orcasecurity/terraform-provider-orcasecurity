package acctest

import (
	"os"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RestoreScmBody builds the PUT body that restores an integrated SCM unit to a
// previously snapshotted state. It runs the snapshot through the resource's
// canonical adopt write path (shift_left_integration.Adopt with an empty plan)
// rather than re-implementing the project_id-XOR-policies rule, so restores
// always follow the exact same contract as a real apply.
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

// APIClient builds an API client from the acceptance-test environment
// variables. It is used by acceptance tests that need to snapshot and restore
// live state (adopt-existing SCM units are not TF-owned, so a test that mutates
// one must put it back).
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
