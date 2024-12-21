package orcasecurity

import (
	"context"
	"fmt"
	"os"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/automations"
	"terraform-provider-orcasecurity/orcasecurity/business_unit"
	"terraform-provider-orcasecurity/orcasecurity/custom_dashboard"
	"terraform-provider-orcasecurity/orcasecurity/custom_discovery_alert"
	"terraform-provider-orcasecurity/orcasecurity/custom_role"
	"terraform-provider-orcasecurity/orcasecurity/custom_sonar_alert"
	"terraform-provider-orcasecurity/orcasecurity/custom_widget"
	"terraform-provider-orcasecurity/orcasecurity/discovery_view"
	"terraform-provider-orcasecurity/orcasecurity/group"
	"terraform-provider-orcasecurity/orcasecurity/jira_template"
	"terraform-provider-orcasecurity/orcasecurity/organizations"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_cve_exception_list"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_project"
	"terraform-provider-orcasecurity/orcasecurity/sonar"
	"terraform-provider-orcasecurity/orcasecurity/trusted_cloud_account"
	"terraform-provider-orcasecurity/orcasecurity/webhooks"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &orcasecurityProvider{}
)

const apiEndpointEnvName = "ORCASECURITY_API_ENDPOINT"
const apiTokenEnvName = "ORCASECURITY_API_TOKEN"

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &orcasecurityProvider{version: version}
	}
}

// orcasecurityProvider is the provider implementation.
type orcasecurityProvider struct {
	version string
}

type orcasecurityProviderModel struct {
	APIEndpoint types.String `tfsdk:"api_endpoint"`
	APIToken    types.String `tfsdk:"api_token"`
}

// Metadata returns the provider type name.
func (p *orcasecurityProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.Version = p.version
	resp.TypeName = "orcasecurity"
}

// Schema defines the provider-level schema for configuration data.
func (p *orcasecurityProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf(`
This provider is used to interact with the resources supported by Orca Security.
The provider needs to be configured with the proper credentials before it can be used.
Use the navigation to the left to get information about the available resources.  

It is required to configure at least two configuration options: api_endpoint and api_token. 
Both can be configured using environment variables "%s" and "%s" respectively.`, apiEndpointEnvName, apiTokenEnvName),
		Attributes: map[string]schema.Attribute{
			"api_endpoint": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: fmt.Sprintf("API endpoint. Alternatively set `%s` environment variable.  ", apiEndpointEnvName) +
					"No default value provided. The provider will not start if none endpoint provided.",
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("API token. Alternatively, set `%s` environment variable.  ", apiTokenEnvName) +
					"Please make sure that API token has enough permissions to access Orca Security resources.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

// Configure prepares a Orca Security API client for data sources and resources.
func (p *orcasecurityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config orcasecurityProviderModel
	diagnostics := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.APIEndpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_endpoint"),
			"Unknown Orca Security API endpoint",
			fmt.Sprintf("The provider cannot create Orca Security API client as there is an unknown configuration value for the Orca Security API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the %s environment variable.",
				apiEndpointEnvName,
			),
		)
	}

	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown Orca Security API token",
			fmt.Sprintf(
				"The provider cannot create Orca Security API client as there is an unknown configuration value for the Orca Security API token. "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the %s environment variable.",
				apiTokenEnvName,
			),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	api_endpoint := os.Getenv(apiEndpointEnvName)
	api_token := os.Getenv(apiTokenEnvName)

	if !config.APIEndpoint.IsNull() {
		api_endpoint = config.APIEndpoint.ValueString()
	}
	if !config.APIToken.IsNull() {
		api_token = config.APIToken.ValueString()
	}

	if api_endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_endpoint"),
			"Missing Orca Security API endpoint",
			fmt.Sprintf("The provider cannot create Orca Security API client as there is a missing or empty value for the Orca Security API endpoint. "+
				"Set the api_endpoint value in the configuration or use %s environment variable. "+
				"If either is already set, ensure the value is not empty.",
				apiEndpointEnvName,
			),
		)
	}
	if api_token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Orca Security API token",
			fmt.Sprintf(
				"The provider cannot create Orca Security API client as there is a missing or empty value for the Orca Security API token. "+
					"Set the api_token value in the configuration or use %s environment variable. "+
					"If either is already set, ensure the value is not empty.",
				apiTokenEnvName,
			),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := api_client.NewAPIClient(&api_endpoint, &api_token)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Orca Security API client",
			"An unexpected error occurred when creating the Orca Security API client. "+
				"If the error is not clear, please contact the provider developers. \n\n"+
				"Orca Security API client error: "+err.Error(),
		)
	}

	resp.ResourceData = client
	resp.DataSourceData = client

	tflog.Info(ctx, fmt.Sprintf("Using %s as Orca Security API base URL", api_endpoint))
}

// DataSources defines the data sources implemented in the provider.
func (p *orcasecurityProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		jira_template.NewJiraTemplateDataSource,
		webhooks.NewWebhookDataSource, organizations.NewOrganizationDataSource,
		sonar.NewSonarQueryDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *orcasecurityProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		automations.NewAutomationResource,
		custom_discovery_alert.NewCustomDiscoveryAlertResource,
		custom_sonar_alert.NewCustomSonarAlertResource,
		custom_role.NewCustomRoleResource,
		group.NewGroupResource,
		business_unit.NewBusinessUnitResource,
		custom_widget.NewCustomWidgetResource,
		custom_dashboard.NewCustomDashboardResource,
		discovery_view.NewDiscoveryViewResource,
		shift_left_project.NewShiftLeftProjectResource,
		shift_left_cve_exception_list.NewShiftLeftCveExceptionListResource,
		trusted_cloud_account.NewTrustedCloudAccountResource,
	}
}
