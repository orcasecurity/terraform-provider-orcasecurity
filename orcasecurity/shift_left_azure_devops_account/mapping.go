package shift_left_azure_devops_account

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func apiToState(inst *api_client.AzureDevopsAccount) resourceModel {
	return resourceModel{
		ID:              types.StringValue(inst.ID),
		InstallationID:  types.StringValue(inst.InstallationID),
		AccountID:       types.StringValue(inst.ID),
		ScmConfigFields: shift_left_integration.ScmConfigFieldsFromAPI(inst.AccountName, inst.ScmUnitCommonFields),
	}
}
