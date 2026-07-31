package shift_left_bitbucket_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewResource() resource.Resource {
	return &shift_left_integration.GenericResource[shift_left_integration.InstallationLifecycle[resourceModel, api_client.BitbucketInstallation]]{
		TypeNameSuffix: "_shift_left_bitbucket_installation",
		SchemaFn:       resourceSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.InstallationLifecycle[resourceModel, api_client.BitbucketInstallation] {
			return shift_left_integration.InstallationLifecycle[resourceModel, api_client.BitbucketInstallation]{
				SCMName: "Bitbucket",
				Create: func(plan *resourceModel) (*api_client.BitbucketInstallation, error) {
					return apiClient.CreateBitbucketInstallation(writeBody(plan))
				},
				Get: apiClient.GetBitbucketInstallation,
				Update: func(plan *resourceModel) (*api_client.BitbucketInstallation, error) {
					return apiClient.UpdateBitbucketInstallation(plan.ID.ValueString(), writeBody(plan))
				},
				Delete:   apiClient.DeleteBitbucketInstallation,
				ID:       func(m *resourceModel) string { return m.ID.ValueString() },
				SetState: setState,
			}
		},
	}
}

type resourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ServerURL         types.String `tfsdk:"server_url"`
	AccessToken       types.String `tfsdk:"access_token"`
	AccessTokenType   types.String `tfsdk:"access_token_type"`
	Username          types.String `tfsdk:"username"`
	AccountID         types.String `tfsdk:"account_id"`
	ExternalServerURL types.String `tfsdk:"external_server_url"`
	IntegrationStatus types.String `tfsdk:"integration_status"`
	CloudIntegration  types.Bool   `tfsdk:"cloud_integration"`
}

func resourceSchema() rschema.Schema {
	attrs := shift_left_integration.InstallationBaseAttrs("Bitbucket", "https://bitbucket.org",
		"Bitbucket access token.")
	attrs["access_token_type"] = shift_left_integration.OptionalComputedString(
		"Token kind: `PAT` for a personal access token, `TOKEN` for a workspace (cloud) or project (server) token.",
		stringvalidator.OneOf("PAT", "TOKEN"))
	attrs["username"] = shift_left_integration.OptionalComputedString(
		"Bitbucket username owning the token (used with `PAT` tokens).")
	attrs["account_id"] = shift_left_integration.OptionalComputedString(
		"Workspace or project slug the installation token is scoped to (`TOKEN` tokens). " +
			"Bitbucket-side slug only — not an Orca UUID, and not the same attribute as " +
			"`orcasecurity_shift_left_bitbucket_account.account_id` (though values often match for workspace tokens).")
	return rschema.Schema{
		Description: "Connects a Bitbucket server or workspace to Orca Shift Left by registering an access token " +
			"(POST /api/shiftleft/bitbucket/installations/). The API never returns the token, so after `terraform import` " +
			"the next apply re-sends the configured token.",
		Attributes: attrs,
	}
}

func writeBody(plan *resourceModel) api_client.BitbucketInstallationWrite {
	return api_client.BitbucketInstallationWrite{
		Name:      plan.Name.ValueString(),
		ServerURL: plan.ServerURL.ValueString(),
		AccessTokenDetails: &api_client.BitbucketAccessTokenDetails{
			AccessToken:     plan.AccessToken.ValueString(),
			AccessTokenType: plan.AccessTokenType.ValueString(),
			Username:        plan.Username.ValueString(),
			AccountID:       plan.AccountID.ValueString(),
		},
	}
}

func setState(m *resourceModel, api *api_client.BitbucketInstallation) {
	m.ID = types.StringValue(api.ID)
	m.Name = types.StringValue(api.Name)
	m.ServerURL = types.StringValue(api.ServerURL)
	m.ExternalServerURL = types.StringValue(api.ExternalServerURL)
	m.IntegrationStatus = types.StringValue(api.IntegrationStatus)
	m.CloudIntegration = types.BoolValue(api.CloudIntegration)
	// access_token is write-only and already stays untouched. The token-echo
	// fields (type/username/account_id) are often null/omitted on read — leave
	// prior state alone when the API is silent so a configured username is not
	// wiped to "" (which then fails the next apply for PAT tokens).
	td := api.AccessTokenDetails
	if td == nil {
		return
	}
	if td.AccessTokenType != "" {
		m.AccessTokenType = types.StringValue(td.AccessTokenType)
	}
	if td.Username != "" {
		m.Username = types.StringValue(td.Username)
	}
	if td.AccountID != "" {
		m.AccountID = types.StringValue(td.AccountID)
	}
}
