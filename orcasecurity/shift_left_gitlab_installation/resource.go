package shift_left_gitlab_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_gitlab_installation",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
			return shift_left_integration.InstallationLifecycle[resourceModel, api_client.GitlabInstallation]{
				SCMName: "GitLab",
				Create: func(plan *resourceModel) (*api_client.GitlabInstallation, error) {
					return apiClient.CreateGitlabInstallation(writeBody(plan))
				},
				Get: apiClient.GetGitlabInstallation,
				Update: func(plan *resourceModel) (*api_client.GitlabInstallation, error) {
					return apiClient.UpdateGitlabInstallation(plan.ID.ValueString(), writeBody(plan))
				},
				Delete:   apiClient.DeleteGitlabInstallation,
				ID:       func(m *resourceModel) string { return m.ID.ValueString() },
				SetState: setState,
			}
		},
	}
}

type resourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	AccessToken       types.String `tfsdk:"access_token"`
	ServerURL         types.String `tfsdk:"server_url"`
	ReadOnly          types.Bool   `tfsdk:"read_only"`
	ExternalServerURL types.String `tfsdk:"external_server_url"`
	AccessTokenName   types.String `tfsdk:"access_token_name"`
	AccessTokenType   types.String `tfsdk:"access_token_type"`
	IntegrationStatus types.String `tfsdk:"integration_status"`
	CloudIntegration  types.Bool   `tfsdk:"cloud_integration"`
}

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.InstallationBaseAttrs("GitLab", "https://gitlab.com",
		"GitLab access token. Orca validates it on create, so it must be a valid group or personal access token.")
	attrs["read_only"] = shift_left_integration.OptionalComputedBool(
		"Whether the token grants read-only access. Defaults to `false`. Must match the actual token permissions.")
	attrs["access_token_name"] = shift_left_integration.ComputedString("Name of the token as reported by GitLab.")
	attrs["access_token_type"] = shift_left_integration.ComputedString("Type of the token as reported by GitLab.")
	return rschema.Schema{
		Description: "Connects a GitLab server to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/gitlab/installations/). Orca validates the token on create, so it must be a valid " +
			"group access token or personal access token. The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
	}
}

func writeBody(plan *resourceModel) api_client.GitlabInstallationWrite {
	return api_client.GitlabInstallationWrite{
		AccessToken: plan.AccessToken.ValueString(),
		Name:        plan.Name.ValueString(),
		ServerURL:   plan.ServerURL.ValueString(),
		// Always sent: the API resets an omitted read_only to false on PATCH.
		ReadOnly: plan.ReadOnly.ValueBool(),
	}
}

func setState(m *resourceModel, api *api_client.GitlabInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.ReadOnly = types.BoolValue(api.ReadOnly)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.AccessTokenName = types.StringValue(api.AccessTokenName)
	m.AccessTokenType = types.StringValue(api.AccessTokenType)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
}
