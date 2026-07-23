package shift_left_azure_devops_account

import (
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type resourceModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	shift_left_integration.ScmConfigFields
}
