package acctest

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// AnchorRuleID is a built-in rule placed in every disposable framework section, so
// that the section survives the alert under test being unlinked from it.
const AnchorRuleID = "rc7bcf3b77f"

// CreateDisposableComplianceFramework creates a custom compliance framework over
// the API and deletes it when the test ends. It returns the framework name, which
// is how the alert resources address it.
//
// Alert acceptance tests need a framework to link into but must not manage one in
// Terraform: the framework resource does not declare the controls that alerts link
// into it, so a managed framework plans those links away and would decide the
// outcome of a test that is about the alert.
func CreateDisposableComplianceFramework(t *testing.T, name string, sections ...string) string {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC=1 to run")
	}

	endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	token := os.Getenv("ORCASECURITY_API_TOKEN")
	client, err := api_client.NewAPIClient(&endpoint, &token)
	if err != nil {
		t.Fatalf("build api client: %s", err)
	}

	request := api_client.CustomComplianceFrameworkRequest{Name: name}
	for index, section := range sections {
		request.Sections = append(request.Sections, api_client.CustomComplianceFrameworkSection{
			Name: section,
			Tests: []api_client.CustomComplianceFrameworkTest{{
				RuleID: AnchorRuleID,
				// One more part than the section's nesting depth, per the API.
				RuleIDInFramework: fmt.Sprintf("%d.1", index+1),
			}},
		})
	}

	created, err := client.CreateCustomComplianceFramework(request)
	if err != nil {
		t.Fatalf("create compliance framework %q: %s", name, err)
	}
	t.Cleanup(func() {
		if err := client.DeleteCustomComplianceFramework(created.ID.String()); err != nil {
			t.Logf("could not delete compliance framework %s: %s", created.ID, err)
		}
	})
	return name
}
