package shift_left_repository

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

// Import cannot set slug (absent from the import ID); Read must backfill it or
// a RequiresReplace attribute stays null and the next plan destroys/recreates.
func TestBitbucketSyncSlug_BackfillsFromAPI(t *testing.T) {
	m := &bitbucketRepositoryModel{}
	bitbucketSyncSlug(m, &api_client.ScmRepository{Slug: "my-repo-slug"})
	if m.Slug.ValueString() != "my-repo-slug" {
		t.Errorf("expected slug backfilled from API, got %q", m.Slug.ValueString())
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

// Each provider's traits must reach its real schema. GitLab is the odd one out at
// the repository level: its skip_check_runs column is constrained to
// GitlabPerformActionStatus (ALWAYS/NEVER), while the other three accept
// ONLY_ON_INTERNAL_ISSUE. Note this differs from the account/group level, where
// all four accept the three-value enum.
func TestProviderSchemas_SkipCheckRunsEnumPerProvider(t *testing.T) {
	cases := []struct {
		name                       string
		res                        resource.Resource
		acceptsOnlyOnInternalIssue bool
	}{
		{"github", NewGithubRepositoryResource(), true},
		{"gitlab", NewGitlabRepositoryResource(), false},
		{"azure", NewAzureDevopsRepositoryResource(), true},
		{"bitbucket", NewBitbucketRepositoryResource(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attr, ok := testutils.ResourceSchemaAttrs(t, tc.res)["skip_check_runs"].(rschema.StringAttribute)
			if !ok {
				t.Fatal("skip_check_runs must be a StringAttribute")
			}
			var diags diag.Diagnostics
			for _, v := range attr.Validators {
				resp := &validator.StringResponse{}
				v.ValidateString(context.Background(), validator.StringRequest{
					Path:        path.Root("skip_check_runs"),
					ConfigValue: types.StringValue("ONLY_ON_INTERNAL_ISSUE"),
				}, resp)
				diags.Append(resp.Diagnostics...)
			}
			if accepted := !diags.HasError(); accepted != tc.acceptsOnlyOnInternalIssue {
				t.Errorf("ONLY_ON_INTERNAL_ISSUE accepted=%v, want %v", accepted, tc.acceptsOnlyOnInternalIssue)
			}

			// ALWAYS is valid everywhere; a bogus value never is.
			for value, wantAccepted := range map[string]bool{"ALWAYS": true, "SOMETIMES": false} {
				var d diag.Diagnostics
				for _, v := range attr.Validators {
					resp := &validator.StringResponse{}
					v.ValidateString(context.Background(), validator.StringRequest{
						Path:        path.Root("skip_check_runs"),
						ConfigValue: types.StringValue(value),
					}, resp)
					d.Append(resp.Diagnostics...)
				}
				if accepted := !d.HasError(); accepted != wantAccepted {
					t.Errorf("%s accepted=%v, want %v", value, accepted, wantAccepted)
				}
			}
		})
	}
}

// skipCheckRunsUnreadable exists only for Azure, whose list serializer omits the
// field. Wiring it to another provider would silently mask real drift.
func TestProviderTraits_OnlyAzureHasUnreadableSkipCheckRuns(t *testing.T) {
	for _, tc := range []struct {
		traits providerTraits
		want   bool
	}{
		{githubTraits, false},
		{gitlabTraits, false},
		{azureTraits, true},
		{bitbucketTraits, false},
	} {
		if tc.traits.skipCheckRunsUnreadable != tc.want {
			t.Errorf("%s skipCheckRunsUnreadable=%v, want %v", tc.traits.name, tc.traits.skipCheckRunsUnreadable, tc.want)
		}
	}
}
