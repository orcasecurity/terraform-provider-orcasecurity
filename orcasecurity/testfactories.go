package orcasecurity

import (
	"os"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const TestProviderConfig = `provider "orcasecurity" {}`

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"orcasecurity": providerserver.NewProtocol6WithError(New("test")()),
}

func TestAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
	if v := os.Getenv("ORCASECURITY_API_ENDPOINT"); v == "" {
		t.Fatal("ORCASECURITY_API_ENDPOINT must be set for acceptance tests")
	}

	if v := os.Getenv("ORCASECURITY_API_TOKEN"); v == "" {
		t.Fatal("ORCASECURITY_API_TOKEN must be set for acceptance tests")
	}
}

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

// TestAPIClient builds an API client from the acceptance-test environment
// variables. It is used by acceptance tests that need to snapshot and restore
// live state (adopt-existing SCM units are not TF-owned, so a test that mutates
// one must put it back).
func TestAPIClient(t *testing.T) *api_client.APIClient {
	t.Helper()
	endpoint := strings.TrimRight(os.Getenv("ORCASECURITY_API_ENDPOINT"), "/")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("failed to create API client for acceptance test: %s", err)
	}
	return client
}
