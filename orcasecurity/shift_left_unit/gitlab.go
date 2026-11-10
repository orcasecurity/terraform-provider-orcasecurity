package shift_left_unit

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var gitlabLabels = shift_left_integration.NewAdoptLabels("GitLab group")

type gitlabGroupModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	GitlabGroupID  types.Int64  `tfsdk:"gitlab_group_id"`
	shift_left_integration.ScmConfigFields
}

// gitlabIntegrateGuard: the GitLab integrate endpoint accepts only ALWAYS/NEVER
// for skip_check_runs; the full enum (ONLY_ON_INTERNAL_ISSUE) is accepted by the
// update endpoint once the group is integrated. Failing here replaces the
// backend's raw validation error with an actionable one.
func gitlabIntegrateGuard(body api_client.ScmInstallationUpdate) error {
	if body.ConfigSettings.SkipCheckRuns != "ONLY_ON_INTERNAL_ISSUE" {
		return nil
	}
	return fmt.Errorf(
		"skip_check_runs = %q is not accepted when integrating a new GitLab group "+
			"(the integrate API allows only ALWAYS or NEVER). Integrate with ALWAYS or NEVER "+
			"and switch to ONLY_ON_INTERNAL_ISSUE on a later apply, or import an "+
			"already-integrated group instead", "ONLY_ON_INTERNAL_ISSUE")
}

func NewGitlabGroupResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_gitlab_group",
		SchemaFn:       gitlabGroupSchema,
		ImportFn:       gitlabImportState,
		OpsFn:          gitlabOps,
	}
}

func gitlabOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.GitlabGroup, gitlabGroupModel]{
		Labels: gitlabLabels,
		UnitID: func(m *gitlabGroupModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return fmt.Sprintf("gitlab_group_id=%d", m.GitlabGroupID.ValueInt64())
		},
		Get: func(m *gitlabGroupModel) (*api_client.GitlabGroup, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return apiClient.GetGitlabGroup(iid, id)
			}
			return apiClient.FindGitlabGroupByGitlabID(iid, m.GitlabGroupID.ValueInt64())
		},
		Update: func(m *gitlabGroupModel, current *api_client.GitlabGroup, body api_client.ScmInstallationUpdate) (*api_client.GitlabGroup, error) {
			return apiClient.UpdateGitlabGroup(m.InstallationID.ValueString(), current.ID, body)
		},
		IntegrateGuard: gitlabIntegrateGuard,
		Integrate: func(m *gitlabGroupModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				GitlabGroupID:  m.GitlabGroupID.ValueInt64(),
				Body:           body,
			})
		},
		Delete:  func(m *gitlabGroupModel) error { return gitlabDeleteGroup(apiClient, m) },
		ToState: gitlabToState,
		Config:  func(m *gitlabGroupModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *gitlabGroupModel) string {
			return fmt.Sprintf("GitLab group %d on installation %q", m.GitlabGroupID.ValueInt64(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca GitLab parent connection first (orcasecurity_shift_left_gitlab_installation).",
		CreateErrorTitle: "Error creating/configuring GitLab group",
		UpdateErrorTitle: "Error updating GitLab group",
		DeleteErrorTitle: "Error deleting GitLab group",
	}
}

// gitlabDeleteGroup resolves Orca id from SCM group id when state lacks id (post-import).
func gitlabDeleteGroup(apiClient *api_client.APIClient, m *gitlabGroupModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.GitlabGroup, error) {
			return apiClient.FindGitlabGroupByGitlabID(m.InstallationID.ValueString(), m.GitlabGroupID.ValueInt64())
		},
		func(g *api_client.GitlabGroup) string { return g.ID },
		func(id string) error { return apiClient.DeleteGitlabGroup(m.InstallationID.ValueString(), id) },
	)
}

func gitlabImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rest, resolved := shift_left_common.ImportScopedInstallation(ctx, req, resp, "<installation_id>/<group_uuid_or_gitlab_group_id>")
	if resolved {
		return
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "right-hand side must be an Orca group UUID or numeric gitlab_group_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("gitlab_group_id"), n)...)
}

func gitlabGroupSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes()
	attrs["account_name"] = shift_left_integration.ComputedAccountName("GitLab group/account name.")
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca GitLab integrated group UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca GitLab installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["gitlab_group_id"] = rschema.Int64Attribute{
		Required:      true,
		Description:   "GitLab-side numeric group ID (from GET .../installations/{id}/groups/).",
		PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca GitLab shift-left integrated group. " +
			"Create POSTs `/api/shiftleft/gitlab/integrated_repositories/` with `group_id`, " +
			"`installation_mode` (defaults to `SELECTED_REPOSITORIES`), and configuration " +
			"(no repositories are attached on that call). If the group is already integrated, " +
			"Create/Update PUT the unit config instead. " +
			"Destroy DELETEs the integrated group (tears down the live integration and its repos). " +
			"Not covered: browse remote groups, check_availability, scan-now (UI operations).",
		Attributes: attrs,
	}
}

func gitlabToState(inst *api_client.GitlabGroup) gitlabGroupModel {
	return gitlabGroupModel{
		ID:              types.StringValue(inst.ID),
		InstallationID:  types.StringValue(inst.InstallationID),
		GitlabGroupID:   types.Int64Value(inst.GitlabGroupID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}

var gitlabGroupsSpec = shift_left_common.ScmObjectListSpec[api_client.GitlabGroup]{
	TypeNameSuffix: "_shift_left_gitlab_groups",
	Description:    "Lists all Orca GitLab shift-left integrated groups for fleet-wide for_each. Use gitlab_group_id with orcasecurity_shift_left_gitlab_group.",
	CollectionKey:  "groups",
	ListErrorTitle: "Error listing GitLab groups",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":              dschema.StringAttribute{Computed: true, Description: "Orca UUID of the GitLab group unit."},
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the parent GitLab installation."},
		"gitlab_group_id": dschema.Int64Attribute{
			Computed:    true,
			Description: "Numeric GitLab group id (from GitLab, not minted by Orca).",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"gitlab_group_id": types.Int64Type,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GitlabGroup, error) {
		return c.ListGitlabGroups()
	},
	Row: func(g *api_client.GitlabGroup) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(g.AccountName, g.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":              types.StringValue(g.ID),
				"installation_id": types.StringValue(g.InstallationID),
				"gitlab_group_id": types.Int64Value(g.GitlabGroupID),
			})
	},
}

func NewGitlabGroupsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(gitlabGroupsSpec)
}
