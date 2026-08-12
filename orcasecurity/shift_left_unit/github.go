package shift_left_unit

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var githubLabels = shift_left_integration.NewAdoptLabels("GitHub account")

type githubAccountModel struct {
	ID types.String `tfsdk:"id"`
	// AccountID is the Orca UUID of the integrated GitHub account. Unlike the other SCMs
	// there is no parent connection resource — the Orca GitHub App installation is itself
	// the account-level unit — so this is the unit's own id and `id` mirrors it.
	AccountID            types.String `tfsdk:"account_id"`
	GithubInstallationID types.Int64  `tfsdk:"github_installation_id"`
	GithubAppSettingsURL types.String `tfsdk:"github_app_settings_url"`
	shift_left_integration.ScmConfigFields
}

func NewGithubAccountResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_github_account",
		SchemaFn:       githubAccountSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("account_id"), req, resp)
		},
		OpsFn: githubOps,
	}
}

func githubOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.GithubInstallation, githubAccountModel]{
		Labels: githubLabels,
		UnitID: func(m *githubAccountModel) string { return m.AccountID.ValueString() },
		Get: func(m *githubAccountModel) (*api_client.GithubInstallation, error) {
			return apiClient.GetGithubInstallation(m.AccountID.ValueString())
		},
		Update: func(m *githubAccountModel, current *api_client.GithubInstallation, body api_client.ScmInstallationUpdate) (*api_client.GithubInstallation, error) {
			return apiClient.UpdateGithubInstallation(current.ID, body)
		},
		// GitHub units come from the App install callback — there is no
		// Integrate POST, so Create always takes the adopt (PUT) path.
		Integrate: nil,
		Delete: func(m *githubAccountModel) error {
			return apiClient.DeleteGithubInstallation(m.AccountID.ValueString())
		},
		ToState: githubToState,
		Config:  func(m *githubAccountModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *githubAccountModel) string {
			return fmt.Sprintf("Account %q", m.AccountID.ValueString())
		},
		CreateHint:       "Install the Orca GitHub App first (UI / GitHub App flow), then import or reference the account_id.",
		CreateErrorTitle: "Error configuring GitHub account",
		UpdateErrorTitle: "Error updating GitHub account",
		DeleteErrorTitle: "Error deleting GitHub account",
	}
}

func githubAccountSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes()
	attrs["account_name"] = shift_left_integration.ComputedAccountName("GitHub account/organization name.")
	attrs["id"] = rschema.StringAttribute{
		Computed: true,
		Description: "Orca UUID of the integrated GitHub account (same value as `account_id`). " +
			"Unlike Bitbucket, GitHub `account_id` is an Orca UUID, not an SCM-side slug.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required: true,
		Description: "Orca UUID of the integrated GitHub account (see `orcasecurity_shift_left_github_accounts`). " +
			"Unlike Bitbucket's `account_id` (an SCM slug), this is an Orca UUID. " +
			"GitHub has no separate installation resource: the Orca GitHub App installation is itself the " +
			"account-level unit, so this is the unit's own id rather than a parent connection id.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["github_installation_id"] = shift_left_common.ComputedInt64(
		"GitHub-side numeric installation ID of the Orca GitHub App.")
	attrs["github_app_settings_url"] = shift_left_common.ComputedString(
		"URL of the Orca GitHub App settings page on GitHub (null when the API omits it).")
	attrs["adopt_existing"] = rschema.BoolAttribute{
		Optional: true,
		Description: "Acknowledge takeover of a unit that already holds state a destroy would drop. GitHub has no " +
			"fresh-integrate path — the account always already exists (from the App install callback) before " +
			"Terraform ever touches it — so unlike the other SCM resources, Create cannot gate this guard on mere " +
			"existence; it would then require this on every apply. It instead refuses only when the account already " +
			"has integrated repositories, attached policies, or a bound project. A later destroy DE-INTEGRATES it " +
			"regardless of what it currently holds. Prefer `terraform import` to bring an existing account under " +
			"management without a takeover write; set this to true only when you intend to manage (and eventually " +
			"tear down) an integration you did not create here.",
	}
	return rschema.Schema{
		Description: "Configures an existing Orca GitHub shift-left account/organization (default policies, scan mode, PR settings). " +
			"This is the GitHub peer of `orcasecurity_shift_left_gitlab_group` and `orcasecurity_shift_left_azure_devops_account`. " +
			"The account must already exist (created by installing the Orca GitHub App — `/github/config/` and App callback are not managed here). " +
			"Create/Update PUT the unit config; Destroy DELETEs the Orca integration (tears down the live integration). " +
			"Not covered: GHES `/github/enterprises/*`, browse repos, check_availability, scan-now. " +
			"Schema follows the Shift-Left API (a superset of the UI): all `configuration_settings` enums are available.",
		Attributes: attrs,
	}
}

func githubToState(account *api_client.GithubInstallation) githubAccountModel {
	return githubAccountModel{
		ID:                   types.StringValue(account.ID),
		AccountID:            types.StringValue(account.ID),
		GithubInstallationID: types.Int64Value(account.GithubInstallationID),
		GithubAppSettingsURL: tfconv.StringOrNull(account.GithubAppSettingsURL),
		ScmConfigFields:      shift_left_integration.ScmConfigFieldsFromAPI(account.AccountName, account.ScmUnitCommonFields),
	}
}

var githubAccountsSpec = shift_left_common.ScmObjectListSpec[api_client.GithubInstallation]{
	TypeNameSuffix: "_shift_left_github_accounts",
	Description: "Lists all Orca GitHub shift-left integrated accounts for fleet-wide for_each. " +
		"account_id is the Orca UUID and mirrors id: GitHub has no separate installation resource, " +
		"so the Orca GitHub App installation is itself the account-level unit.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing GitHub accounts",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":         dschema.StringAttribute{Computed: true, Description: "Orca UUID of the GitHub account unit."},
		"account_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the account (mirrors `id`; GitHub has no separate installation resource)."},
		"github_installation_id": dschema.Int64Attribute{
			Computed:    true,
			Description: "Numeric GitHub App installation id (from GitHub, not minted by Orca).",
		},
		"github_app_settings_url": dschema.StringAttribute{
			Computed:    true,
			Description: "GitHub URL for managing the App installation, when reported.",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":                      types.StringType,
		"account_id":              types.StringType,
		"github_installation_id":  types.Int64Type,
		"github_app_settings_url": types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.GithubInstallation, error) {
		return c.ListGithubInstallations()
	},
	Row: func(a *api_client.GithubInstallation) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":                      types.StringValue(a.ID),
				"account_id":              types.StringValue(a.ID),
				"github_installation_id":  types.Int64Value(a.GithubInstallationID),
				"github_app_settings_url": tfconv.StringOrNull(a.GithubAppSettingsURL),
			})
	},
}

func NewGithubAccountsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(githubAccountsSpec)
}
