package jira_cloud_resource

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
	_ datasource.DataSource              = &jiraCloudDataSource{}
	_ datasource.DataSourceWithConfigure = &jiraCloudDataSource{}
)

type jiraCloudDataSource struct {
	apiClient *api_client.APIClient
}

type jiraCloudDataSourceModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
	URL  types.String `tfsdk:"url"`
}

func NewJiraCloudDataSource() datasource.DataSource {
	return &jiraCloudDataSource{}
}

func (ds *jiraCloudDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_jira_cloud_resource"
}

func (ds *jiraCloudDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *jiraCloudDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up a connected Jira Cloud site (resource) by name. Jira Cloud credentials are created via the OAuth flow in the Orca UI, so this is read-only. Use the returned `id` as `resource_id` on `orcasecurity_integration_jira_cloud_template`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the connected Jira Cloud site (its subdomain / display name in Orca).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Jira Cloud resource identifier (cloud id). Use it as `resource_id` on the Jira Cloud template.",
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Jira Cloud site.",
			},
		},
	}
}

func (ds *jiraCloudDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state jiraCloudDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := ds.apiClient.GetJiraCloudResourceByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error looking up Jira Cloud resource",
			fmt.Sprintf("Could not look up Jira Cloud resource %q: %s", state.Name.ValueString(), err.Error()),
		)
		return
	}
	if current == nil {
		resp.Diagnostics.AddError(
			"Jira Cloud resource not found",
			fmt.Sprintf("No connected Jira Cloud resource named %q was found. Connect Jira Cloud in the Orca UI first.", state.Name.ValueString()),
		)
		return
	}

	state.ID = types.StringValue(current.ID)
	state.URL = types.StringValue(current.URL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
