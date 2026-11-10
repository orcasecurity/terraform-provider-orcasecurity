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

var azureLabels = shift_left_integration.NewAdoptLabels("Azure DevOps account")

type azureAccountModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	shift_left_integration.ScmConfigFields
}

func NewAzureDevopsAccountResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_azure_devops_account",
		SchemaFn:       azureAccountSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			shift_left_common.ImportScopedUnit(ctx, req, resp, "account_name", "<installation_id>/<account_name_or_orca_uuid>")
		},
		OpsFn: azureOps,
	}
}

func azureOps(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
	return shift_left_integration.AdoptedUnitOps[api_client.AzureDevopsAccount, azureAccountModel]{
		Labels: azureLabels,
		UnitID: func(m *azureAccountModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return m.AccountName.ValueString()
		},
		Get: func(m *azureAccountModel) (*api_client.AzureDevopsAccount, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return apiClient.GetAzureDevopsAccount(iid, id)
			}
			return apiClient.FindAzureDevopsAccountByName(iid, m.AccountName.ValueString())
		},
		Update: func(m *azureAccountModel, current *api_client.AzureDevopsAccount, body api_client.ScmInstallationUpdate) (*api_client.AzureDevopsAccount, error) {
			return apiClient.UpdateAzureDevopsAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *azureAccountModel, body api_client.ScmInstallationUpdate) error {
			return apiClient.IntegrateAzureDevopsUnit(api_client.AzureDevopsUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountName:    m.AccountName.ValueString(),
				Body:           body,
			})
		},
		Delete:  func(m *azureAccountModel) error { return azureDeleteAccount(apiClient, m) },
		ToState: azureToState,
		Config:  func(m *azureAccountModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *azureAccountModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountName.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca Azure DevOps parent connection first (orcasecurity_shift_left_azure_devops_installation).",
		CreateErrorTitle: "Error creating/configuring Azure DevOps account",
		UpdateErrorTitle: "Error updating Azure DevOps account",
		DeleteErrorTitle: "Error deleting Azure DevOps account",
	}
}

// azureDeleteAccount resolves Orca id from account name when state lacks id (post-import).
func azureDeleteAccount(apiClient *api_client.APIClient, m *azureAccountModel) error {
	return shift_left_integration.DeleteByLookup(
		m.ID.ValueString(),
		func() (*api_client.AzureDevopsAccount, error) {
			return apiClient.FindAzureDevopsAccountByName(m.InstallationID.ValueString(), m.AccountName.ValueString())
		},
		func(a *api_client.AzureDevopsAccount) string { return a.ID },
		func(id string) error { return apiClient.DeleteAzureDevopsAccount(m.InstallationID.ValueString(), id) },
	)
}

func azureAccountSchema() rschema.Schema {
	attrs := shift_left_integration.SharedScmConfigAttributes()
	// Unlike the other SCMs, account_name is the identity here (Required), not a
	// server-reported display name.
	attrs["account_name"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Azure DevOps organization name (API `azure_account_name` on integrate).",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	attrs["id"] = rschema.StringAttribute{
		Computed:      true,
		Description:   "Orca Azure DevOps integrated account UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
	attrs["installation_id"] = rschema.StringAttribute{
		Required:      true,
		Description:   "Orca Azure DevOps installation UUID.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
	}
	return rschema.Schema{
		Description: "Creates or configures an Orca Azure DevOps shift-left integrated account (organization). " +
			"Create POSTs `/api/shiftleft/azure_devops/integrated_repositories/` with `azure_account_name`, " +
			"`installation_mode` (defaults to `SELECTED_REPOSITORIES`), and configuration " +
			"(no repositories are attached on that call). " +
			"If already integrated, Create/Update PUT the unit config. Destroy DELETEs the integrated account. " +
			"Not covered: browse accounts, check_availability, scan-now. " +
			"Schema follows the Shift-Left API (a superset of the Azure UI, which hides skip_check_runs and archive actions).",
		Attributes: attrs,
	}
}

func azureToState(inst *api_client.AzureDevopsAccount) azureAccountModel {
	return azureAccountModel{
		ID:              types.StringValue(inst.ID),
		InstallationID:  types.StringValue(inst.InstallationID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}

var azureAccountsSpec = shift_left_common.ScmObjectListSpec[api_client.AzureDevopsAccount]{
	TypeNameSuffix: "_shift_left_azure_devops_accounts",
	Description:    "Lists all Orca Azure DevOps shift-left integrated accounts for fleet-wide for_each. Use account_name with orcasecurity_shift_left_azure_devops_account.",
	CollectionKey:  "accounts",
	ListErrorTitle: "Error listing Azure DevOps accounts",
	Attrs: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrs(), map[string]dschema.Attribute{
		"id":              dschema.StringAttribute{Computed: true, Description: "Orca UUID of the Azure DevOps account unit."},
		"installation_id": dschema.StringAttribute{Computed: true, Description: "Orca UUID of the parent Azure DevOps installation."},
	}),
	AttrTypes: shift_left_common.MergeMaps(shift_left_integration.SharedScmListUnitAttrTypes(), map[string]attr.Type{
		"id":              types.StringType,
		"installation_id": types.StringType,
	}),
	List: func(c *api_client.APIClient) ([]api_client.AzureDevopsAccount, error) {
		return c.ListAzureDevopsAccounts()
	},
	Row: func(a *api_client.AzureDevopsAccount) map[string]attr.Value {
		return shift_left_common.MergeMaps(
			shift_left_integration.SharedScmListUnitValues(a.AccountName, a.ScmUnitCommonFields),
			map[string]attr.Value{
				"id":              types.StringValue(a.ID),
				"installation_id": types.StringValue(a.InstallationID),
			})
	},
}

func NewAzureDevopsAccountsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(azureAccountsSpec)
}
