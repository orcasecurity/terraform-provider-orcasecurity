package orcasecurity

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &jiraTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &jiraTemplateDataSource{}
)

type jiraTemplateStateModel struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
}

type jiraTemplateDataSource struct {
	apiClient *api_client.APIClient
}

func NewJiraTemplateDataSource() datasource.DataSource {
	return &jiraTemplateDataSource{}
}

func (ds *jiraTemplateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *jiraTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jira_template"
}

func (ds *jiraTemplateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch Jira template data.",
		Attributes: map[string]schema.Attribute{
			"jira_template": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id":            schema.StringAttribute{},
					"template_name": schema.StringAttribute{},
				},
			},
		},
	}
}

func (ds *jiraTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state jiraTemplateStateModel

	item, err := ds.apiClient.GetJiraTemplate(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Jira templates", err.Error())
		return
	}

	state.ID = types.StringValue(item.ID)
	state.TemplateName = types.StringValue(item.TemplateName)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
