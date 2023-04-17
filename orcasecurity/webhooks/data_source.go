package webhooks

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dataSource{}
	_ datasource.DataSourceWithConfigure = &dataSource{}
)

type stateModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type dataSource struct {
	apiClient *api_client.APIClient
}

func NewWebhookDataSource() datasource.DataSource {
	return &dataSource{}
}

func (ds *dataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *dataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (ds *dataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch Web hook.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Web hook name at Orca",
			},
		},
	}
}

func (ds *dataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state stateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	item, err := ds.apiClient.GetWebhookByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Web hooks", err.Error())
		return
	}

	state.ID = types.StringValue(item.ID)
	state.Name = types.StringValue(item.TemplateName)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
