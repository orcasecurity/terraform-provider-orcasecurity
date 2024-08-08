package azure_devops_template

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &azureDevopsTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &azureDevopsTemplateDataSource{}
)

type azureDevopsTemplateStateModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type azureDevopsTemplateDataSource struct {
	apiClient *api_client.APIClient
}

func NewAzureDevopsTemplateDataSource() datasource.DataSource {
	return &azureDevopsTemplateDataSource{}
}

func (ds *azureDevopsTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *azureDevopsTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azure_devops_template"
}

func (ds *azureDevopsTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch Azure Devops template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Template name at Orca",
			},
		},
	}
}

func (ds *azureDevopsTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state azureDevopsTemplateStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	item, err := ds.apiClient.GetAzureDevopsTemplateByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Azure DevOps templates", err.Error())
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
