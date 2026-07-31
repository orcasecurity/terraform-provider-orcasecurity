package shift_left_repository

import (
	"context"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type bitbucketRepositoryModel struct {
	InstallationID        types.String `tfsdk:"installation_id"`
	AccountID             types.String `tfsdk:"account_id"`
	BitbucketRepositoryID types.String `tfsdk:"bitbucket_repository_id"`
	Slug                  types.String `tfsdk:"slug"`
	RepoConfigFields
}

func NewBitbucketRepositoryResource() resource.Resource {
	return NewRepoResource(RepoSpec[bitbucketRepositoryModel]{
		TypeNameSuffix: "_shift_left_bitbucket_repository",
		SchemaFn:       bitbucketRepositorySchema,
		ImportFn:       bitbucketRepositoryImportState,
		Ops:            bitbucketRepositoryOps,
		Fields:         (*bitbucketRepositoryModel).Fields,
	})
}

func bitbucketRepositorySchema() rschema.Schema {
	attrs := sharedRepoAttributes(bitbucketTraits)
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca id of the Bitbucket installation (see `orcasecurity_shift_left_bitbucket_installation`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required: true,
		Description: "Bitbucket workspace slug (cloud) or project key (server) owning the repository. " +
			"Bitbucket-side identity — not an Orca UUID (see `orcasecurity_shift_left_bitbucket_accounts.account_id`).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["bitbucket_repository_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Bitbucket repository id.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["slug"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Bitbucket repository slug.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Integrates a single Bitbucket repository into Orca Shift Left under an existing Bitbucket installation. " +
			"Destroying the resource un-integrates the repository (deletes its repository context); it does not touch the repository on Bitbucket. " +
			"Import with `installation_id:account_id:bitbucket_repository_id`.",
		Attributes: attrs,
	}
}

func bitbucketRepositoryOps(apiClient *api_client.APIClient, plan *bitbucketRepositoryModel) repoOps {
	installationID := plan.InstallationID.ValueString()
	accountID := plan.AccountID.ValueString()
	repoID := plan.BitbucketRepositoryID.ValueString()
	return repoOps{
		client: apiClient,
		traits: bitbucketTraits,
		integrate: func() error {
			return apiClient.IntegrateBitbucketRepository(api_client.BitbucketRepositoryIntegrate{
				InstallationID:        plan.InstallationID.ValueString(),
				AccountID:             accountID,
				BitbucketRepositoryID: repoID,
				Slug:                  plan.Slug.ValueString(),
				Name:                  plan.Name.ValueString(),
				URL:                   plan.URL.ValueString(),
				Branch:                plan.Branch.ValueString(),
				ProjectID:             plan.ProjectID.ValueString(),
				Config:                integrateConfig(&plan.RepoConfigFields),
			})
		},
		syncIdentity: func(row *api_client.ScmRepository) { bitbucketSyncSlug(plan, row) },
		find: func() (*api_client.ScmRepository, error) {
			return apiClient.FindBitbucketRepository(installationID, accountID, repoID)
		},
		update: apiClient.UpdateBitbucketRepositories,
	}
}

// Import cannot set slug (Required+RequiresReplace) — backfill on read or import plans destroy/recreate.
// Only overwrite from a non-empty value: slug forces replacement, so writing "" for a row whose slug the
// list API omitted would propose destroying a healthy integration.
func bitbucketSyncSlug(m *bitbucketRepositoryModel, row *api_client.ScmRepository) {
	if row.Slug == "" {
		return
	}
	m.Slug = types.StringValue(row.Slug)
}

func bitbucketRepositoryImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "expected format installation_id:account_id:bitbucket_repository_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bitbucket_repository_id"), parts[2])...)
}
