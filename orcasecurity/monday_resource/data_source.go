package monday_resource

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
	_ datasource.DataSource              = &mondayDataSource{}
	_ datasource.DataSourceWithConfigure = &mondayDataSource{}
)

type mondayDataSource struct {
	apiClient *api_client.APIClient
}

type mondayDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	AccountSlug types.String `tfsdk:"account_slug"`
}

func NewMondayDataSource() datasource.DataSource {
	return &mondayDataSource{}
}

func (ds *mondayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_monday_resource"
}

func (ds *mondayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *mondayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Monday.com credentials resource by name. Use the returned `id` as `resource_id` on `orcasecurity_integration_monday_template` when the credentials side was created outside Terraform (or in a different workspace).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-friendly name of the existing Monday.com resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca resource identifier (UUID).",
			},
			"account_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Monday.com account slug stored on the resource.",
			},
		},
	}
}

func (ds *mondayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state mondayDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := ds.apiClient.GetMondayResourceByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error looking up Monday resource",
			fmt.Sprintf("Could not look up Monday resource %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.Diagnostics.AddError(
			"Monday resource not found",
			fmt.Sprintf("No Monday resource named %q was found in this organisation.", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.StringValue(current.ID)
	state.AccountSlug = types.StringValue(current.Data.AccountSlug)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
