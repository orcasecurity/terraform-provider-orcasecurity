package servicenow

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &itsmDataSource{}
	_ datasource.DataSourceWithConfigure = &itsmDataSource{}
)

type itsmDataSource struct {
	apiClient *api_client.APIClient
}

type itsmDataSourceModel struct {
	Name          types.String `tfsdk:"name"`
	ID            types.String `tfsdk:"id"`
	ServiceNowURL types.String `tfsdk:"servicenow_url"`
	Username      types.String `tfsdk:"username"`
}

func NewServiceNowDataSource() datasource.DataSource {
	return &itsmDataSource{}
}

func (ds *itsmDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_servicenow_resource"
}

func (ds *itsmDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *itsmDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing ServiceNow ITSM credentials resource by name. Use the returned `id` as `resource_id` on `orcasecurity_integration_servicenow_itsm_template` when the credentials side of the integration was created outside Terraform (or in a different workspace).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-friendly name of the existing ServiceNow ITSM resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca resource identifier (UUID).",
			},
			"servicenow_url": schema.StringAttribute{
				Computed:    true,
				Description: "ServiceNow instance URL stored on the resource.",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "ServiceNow username stored on the resource.",
			},
		},
	}
}

func (ds *itsmDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state itsmDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := ds.apiClient.GetServiceNowITSMResourceByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error looking up ServiceNow ITSM resource",
			fmt.Sprintf("Could not look up ServiceNow ITSM resource %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.Diagnostics.AddError(
			"ServiceNow ITSM resource not found",
			fmt.Sprintf("No ServiceNow ITSM resource named %q was found in this organisation.", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.StringValue(current.ID)
	state.ServiceNowURL = types.StringValue(current.HostURL)
	state.Username = types.StringValue(current.Data.Username)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
