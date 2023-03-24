package orcasecurity

import (
	"context"
	"os"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &orcasecurityProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New() provider.Provider {
	return &orcasecurityProvider{}
}

// orcasecurityProvider is the provider implementation.
type orcasecurityProvider struct{}

type orcasecurityProviderModel struct {
	APIEndpoint types.String `tfsdk:"api_endpoint"`
	APIToken    types.String `tfsdk:"api_token"`
}

// Metadata returns the provider type name.
func (p *orcasecurityProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "orcasecurity"
}

// Schema defines the provider-level schema for configuration data.
func (p *orcasecurityProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_endpoint": schema.StringAttribute{Optional: true},
			"api_token":    schema.StringAttribute{Optional: true, Sensitive: true},
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
			"The provider cannot create Orca Security API client as there is an unknown configuration value for the Orca Security API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ORCASECURITY_API_ENDPOINT environment variable.",
		)
	}

	if config.APIToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown Orca Security API token",
			"The provider cannot create Orca Security API client as there is an unknown configuration value for the Orca Security API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the ORCASECURITY_API_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	api_endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
	api_token := os.Getenv("ORCASECURITY_API_TOKEN")

	if !config.APIEndpoint.IsNull() {
		api_endpoint = config.APIEndpoint.ValueString()
	}
	if !config.APIToken.IsNull() {
		api_token = config.APIEndpoint.ValueString()
	}

	if api_endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_endpoint"),
			"Missing Orca Security API endpoint",
			"The provider cannot create Orca Security API client as there is a missing or empty value for the Orca Security API endpoint. "+
				"Set the api_endpoint value in the configuration or use ORCASECURITY_API_ENDPOINT environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if api_token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Orca Security API endpoint",
			"The provider cannot create Orca Security API client as there is a missing or empty value for the Orca Security API endpoint. "+
				"Set the api_token value in the configuration or use ORCASECURITY_API_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
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

}

// DataSources defines the data sources implemented in the provider.
func (p *orcasecurityProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCoffeesDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *orcasecurityProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}
