package shift_left_repository

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// importErr invokes a resource's ImportState with the given import ID and
// returns whether it produced an error diagnostic. Only the malformed-ID
// branches are exercised: a well-formed ID falls through to
// resp.State.SetAttribute, which panics without a schema-backed state, so the
// happy path is left to the acceptance import tests. Every case here is expected
// to error before reaching SetAttribute.
func importErr(t *testing.T, r resource.ResourceWithImportState, id string) bool {
	t.Helper()
	resp := &resource.ImportStateResponse{}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, resp)
	return resp.Diagnostics.HasError()
}

func TestGithubImportState_RejectsMalformedID(t *testing.T) {
	r := &githubRepositoryResource{}
	for _, id := range []string{
		"",                  // no separator
		"inst-1",            // no separator
		"inst-1:notanumber", // non-numeric repo id
	} {
		if !importErr(t, r, id) {
			t.Errorf("github import %q should have errored", id)
		}
	}
}

func TestGitlabImportState_RejectsMalformedID(t *testing.T) {
	r := &gitlabRepositoryResource{}
	for _, id := range []string{
		"inst-1:7",     // too few parts
		"inst-1:7:8:9", // too many parts
		"inst-1:x:8",   // non-numeric group
		"inst-1:7:y",   // non-numeric project
	} {
		if !importErr(t, r, id) {
			t.Errorf("gitlab import %q should have errored", id)
		}
	}
}

func TestAzureImportState_RejectsMalformedID(t *testing.T) {
	r := &azureRepositoryResource{}
	for _, id := range []string{
		"a:b:c",     // too few parts
		"a:b:c:d:e", // too many parts
	} {
		if !importErr(t, r, id) {
			t.Errorf("azure import %q should have errored", id)
		}
	}
}

func TestBitbucketImportState_RejectsMalformedID(t *testing.T) {
	r := &bitbucketRepositoryResource{}
	for _, id := range []string{
		"a:b",     // too few parts
		"a:b:c:d", // too many parts
	} {
		if !importErr(t, r, id) {
			t.Errorf("bitbucket import %q should have errored", id)
		}
	}
}

// The provider Schema methods wire branch-required and provider-specific
// attributes; assert that wiring rather than re-testing sharedRepoAttributes.
func TestProviderSchemas_BranchRequirednessAndKeys(t *testing.T) {
	cases := []struct {
		name           string
		res            resource.Resource
		branchRequired bool
		extraKeys      []string
	}{
		{"github", NewGithubRepositoryResource(), true, []string{"installation_id", "github_repository_id"}},
		{"gitlab", NewGitlabRepositoryResource(), true, []string{"installation_id", "gitlab_group_id", "gitlab_project_id"}},
		{"azure", NewAzureDevopsRepositoryResource(), false, []string{"installation_id", "account_name", "azure_repository_id", "azure_project_id"}},
		{"bitbucket", NewBitbucketRepositoryResource(), false, []string{"installation_id", "account_id", "bitbucket_repository_id", "slug"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := testutils.ResourceSchemaAttrs(t, tc.res)
			branch, ok := attrs["branch"]
			if !ok {
				t.Fatal("branch attribute missing")
			}
			if branch.IsRequired() != tc.branchRequired {
				t.Errorf("branch Required=%v, want %v", branch.IsRequired(), tc.branchRequired)
			}
			for _, k := range tc.extraKeys {
				if _, ok := attrs[k]; !ok {
					t.Errorf("missing provider attribute %q", k)
				}
			}
		})
	}
}
