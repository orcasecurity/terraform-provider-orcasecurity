package shift_left_unit

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var bitbucketLabels = shift_left_integration.NewAdoptLabels("Bitbucket account")

type bitbucketAccountModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	AccountID      types.String `tfsdk:"account_id"`
	shift_left_integration.ScmConfigFields
}

// bitbucketIntegrateGuard: the Bitbucket integrate endpoint requires an explicit
// repositories list under SELECTED_REPOSITORIES and rejects the request
// without one — and this resource intentionally does not model per-repository
// selection (that is orcasecurity_shift_left_bitbucket_repository). Failing
// here replaces the backend's raw validation error with an actionable one.
// Only integrate is affected: updating an existing account to
// SELECTED_REPOSITORIES is accepted.
func bitbucketIntegrateGuard(body api_client.ScmInstallationUpdate) error {
	if body.InstallationMode != "SELECTED_REPOSITORIES" {
		return nil
	}
	return fmt.Errorf(
		"integrating a new Bitbucket workspace requires installation_mode = %q: "+
			"the API only accepts SELECTED_REPOSITORIES on integrate together with an explicit "+
			"repository list, which this resource does not send. Integrate with SCAN_ALL_INCLUDE_FUTURE "+
			"(you can switch to SELECTED_REPOSITORIES on a later apply), or import an already-integrated "+
			"account instead", "SCAN_ALL_INCLUDE_FUTURE")
}

func NewBitbucketAccountResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_bitbucket_account",
		SchemaFn:       bitbucketAccountSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			shift_left_common.ImportScopedUnit(ctx, req, resp, "account_id", "<installation_id>/<account_slug_or_orca_uuid>")
		},
		OpsFn: bitbucketOps,
	}
}

func bitbucketOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, bitbucketAccountModel]{
		Labels: bitbucketLabels,
		UnitID: func(m *bitbucketAccountModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return m.AccountID.ValueString()
		},
		Get: func(m *bitbucketAccountModel) (*api_client.BitbucketAccount, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return apiClient.GetBitbucketAccount(iid, id)
			}
			return apiClient.FindBitbucketAccountBySlug(iid, m.AccountID.ValueString())
		},
		Update: func(m *bitbucketAccountModel, current *api_client.BitbucketAccount, body api_client.ScmInstallationUpdate) (*api_client.BitbucketAccount, error) {
			return apiClient.UpdateBitbucketAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		IntegrateGuard: bitbucketIntegrateGuard,
		Integrate: func(m *bitbucketAccountModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountID:      m.AccountID.ValueString(),
				Body:           body,
			})
		},
		Delete:  func(m *bitbucketAccountModel) error { return bitbucketDeleteAccount(apiClient, m) },
		ToState: bitbucketToState,
		Config:  func(m *bitbucketAccountModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *bitbucketAccountModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca Bitbucket parent connection first (orcasecurity_shift_left_bitbucket_installation).",
		CreateErrorTitle: "Error creating/configuring Bitbucket account",
		UpdateErrorTitle: "Error updating Bitbucket account",
		DeleteErrorTitle: "Error deleting Bitbucket account",
	}
}

// bitbucketDeleteAccount resolves Orca id from account slug when state lacks id (post-import).
func bitbucketDeleteAccount(apiClient *api_client.APIClient, m *bitbucketAccountModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.BitbucketAccount, error) {
			return apiClient.FindBitbucketAccountBySlug(m.InstallationID.ValueString(), m.AccountID.ValueString())
		},
		func(a *api_client.BitbucketAccount) string { return a.ID },
		func(id string) error { return apiClient.DeleteBitbucketAccount(m.InstallationID.ValueString(), id) },
	)
}

func bitbucketAccountSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes()
	attrs["account_name"] = shift_left_integration.ComputedAccountName("Bitbucket workspace/account name.")
	// The backend has no Bitbucket scope on SCM posture policies, so the shared
	// attribute can never carry a value here.
	attrs["scm_posture_policy_id"] = shift_left_integration.ComputedVolatileString(
		"Always null for Bitbucket: SCM posture policies cannot scope Bitbucket workspaces, so no posture policy ever attaches to this unit. Present for schema parity with the other SCM account resources.",
	)
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca Bitbucket integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Bitbucket installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["account_id"] = rschema.StringAttribute{
		Required: true,
		Description: "Bitbucket-side workspace slug (cloud) or project key (server). " +
			"This is NOT an Orca UUID — the Orca unit id is the computed `id`. " +
			"Do not confuse with `orcasecurity_shift_left_bitbucket_installation.account_id` " +
			"(token-scope slug on the installation credential).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca Bitbucket shift-left integrated account. " +
			"Create POSTs `/api/shiftleft/bitbucket/integrated_repositories/` with Bitbucket `account_id` (slug), " +
			"`installation_mode`, and configuration (no repositories are attached on that call). " +
			"Integrating a not-yet-integrated workspace requires `installation_mode = \"SCAN_ALL_INCLUDE_FUTURE\"`: " +
			"the API accepts `SELECTED_REPOSITORIES` on integrate only together with an explicit repository list, " +
			"which this resource does not send (you can switch modes on a later apply). " +
			"If already integrated, Create/Update PUT the unit config. Destroy DELETEs the integrated account. " +
			"Not covered: browse accounts, check_availability, scan-now. " +
			"Archive/unavailable actions in configuration_settings may be ignored by the Bitbucket API.",
		Attributes: attrs,
	}
}

func bitbucketToState(inst *api_client.BitbucketAccount) bitbucketAccountModel {
	return bitbucketAccountModel{
		ID:              types.StringValue(inst.ID),
		InstallationID:  types.StringValue(inst.InstallationID),
		AccountID:       types.StringValue(inst.AccountID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}

var bitbucketAccountsSpec = shift_left_common.ScmObjectListSpec[api_client.BitbucketAccount]{
	TypeNameSuffix: "_shift_left_bitbucket_accounts",
	Description: "Lists all Orca Bitbucket shift-left integrated accounts for fleet-wide for_each. " +
		"`account_id` is the Bitbucket workspace slug (not an Orca UUID); `id` is the Orca unit UUID; " +
		"`installation_id` is the parent Orca Bitbucket installation UUID.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing Bitbucket accounts",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":              dschema.StringAttribute{Computed: true, Description: "Orca UUID of the Bitbucket account unit."},
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the parent Bitbucket installation."},
		"account_id": dschema.StringAttribute{
			Computed:    true,
			Description: "Bitbucket workspace slug (cloud) or project key (server); not an Orca UUID.",
		},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
		"account_id":      types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.BitbucketAccount, error) {
		return c.ListBitbucketAccounts()
	},
	Row: func(a *api_client.BitbucketAccount) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":              types.StringValue(a.ID),
				"installation_id": types.StringValue(a.InstallationID),
				"account_id":      types.StringValue(a.AccountID),
			})
	},
}

func NewBitbucketAccountsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(bitbucketAccountsSpec)
}
