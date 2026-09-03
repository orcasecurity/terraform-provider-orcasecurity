package shift_left_installation

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type bitbucketInstallationModel struct {
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

func NewBitbucketInstallationResource() resource.Resource {
	return &shift_left_integration.GenericResource{
		TypeNameSuffix: "_shift_left_bitbucket_installation",
		SchemaFn:       bitbucketInstallationSchema,
		ImportFn: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
		},
		OpsFn: func(apiClient *api_client.APIClient) shift_left_integration.UnitOps {
			return shift_left_integration.InstallationLifecycle[bitbucketInstallationModel, api_client.BitbucketInstallation]{
				SCMName: "Bitbucket",
				Create: func(plan *bitbucketInstallationModel) (*api_client.BitbucketInstallation, error) {
					return apiClient.CreateBitbucketInstallation(bitbucketWriteBody(plan))
				},
				Get: apiClient.GetBitbucketInstallation,
				Update: func(plan *bitbucketInstallationModel) (*api_client.BitbucketInstallation, error) {
					return apiClient.UpdateBitbucketInstallation(plan.ID.ValueString(), bitbucketWriteBody(plan))
				},
				Delete:   apiClient.DeleteBitbucketInstallation,
				ID:       func(m *bitbucketInstallationModel) string { return m.ID.ValueString() },
				SetState: bitbucketSetState,
			}
		},
	}
}

func bitbucketInstallationSchema() rschema.Schema {
	attrs := shift_left_integration.InstallationBaseAttrs("Bitbucket", "https://bitbucket.org",
		"Bitbucket access token.")
	attrs["access_token_type"] = shift_left_common.OptionalComputedString(
		"Token kind: `PAT` for a personal access token, `TOKEN` for a workspace (cloud) or project (server) token.",
		stringvalidator.OneOf("PAT", "TOKEN"))
	attrs["username"] = shift_left_common.OptionalComputedString(
		"Bitbucket username owning the token (used with `PAT` tokens).")
	attrs["account_id"] = shift_left_common.OptionalComputedString(
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

func bitbucketWriteBody(plan *bitbucketInstallationModel) api_client.BitbucketInstallationWrite {
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

func bitbucketSetState(m *bitbucketInstallationModel, api *api_client.BitbucketInstallation) {
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

var bitbucketInstallationsSpec = shift_left_common.ScmObjectListSpec[api_client.BitbucketInstallation]{
	TypeNameSuffix: "_shift_left_bitbucket_installations",
	Description: "Lists all Orca Bitbucket shift-left server connections (installations) for fleet-wide for_each. " +
		"Use each installation's `id` as `installation_id` on `orcasecurity_shift_left_bitbucket_account` / repository resources. " +
		"`account_id` here is the token-scope workspace/project slug from the installation credential — " +
		"not the Orca account UUID and not necessarily the same field as on `orcasecurity_shift_left_bitbucket_account`.",
	CollectionKey:  "installations",
	ListErrorTitle: "Error listing Bitbucket installations",
	Attrs: map[string]dschema.Attribute{
		"id":                  dschema.StringAttribute{Computed: true, Description: "Orca Bitbucket installation UUID."},
		"name":                dschema.StringAttribute{Computed: true},
		"server_url":          dschema.StringAttribute{Computed: true},
		"external_server_url": dschema.StringAttribute{Computed: true},
		"access_token_type":   dschema.StringAttribute{Computed: true},
		"username":            dschema.StringAttribute{Computed: true, Description: "Bitbucket username owning a PAT token when present."},
		"account_id": dschema.StringAttribute{
			Computed: true,
			Description: "Token-scope workspace/project slug from the installation credential (`TOKEN` tokens). " +
				"This is a Bitbucket-side slug, not an Orca UUID.",
		},
		"integration_status": dschema.StringAttribute{Computed: true},
		"cloud_integration":  dschema.BoolAttribute{Computed: true},
	},
	AttrTypes: map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "server_url": types.StringType,
		"external_server_url": types.StringType, "access_token_type": types.StringType,
		"username": types.StringType, "account_id": types.StringType,
		"integration_status": types.StringType, "cloud_integration": types.BoolType,
	},
	List: func(c *api_client.APIClient) ([]api_client.BitbucketInstallation, error) {
		return c.ListBitbucketInstallations()
	},
	Row: func(a *api_client.BitbucketInstallation) map[string]attr.Value {
		tokenType, username, accountID := "", "", ""
		if a.AccessTokenDetails != nil {
			tokenType = a.AccessTokenDetails.AccessTokenType
			username = a.AccessTokenDetails.Username
			accountID = a.AccessTokenDetails.AccountID
		}
		return map[string]attr.Value{
			"id":                  types.StringValue(a.ID),
			"name":                types.StringValue(a.Name),
			"server_url":          tfconv.StringOrNull(a.ServerURL),
			"external_server_url": tfconv.StringOrNull(a.ExternalServerURL),
			"access_token_type":   tfconv.StringOrNull(tokenType),
			"username":            tfconv.StringOrNull(username),
			"account_id":          tfconv.StringOrNull(accountID),
			"integration_status":  tfconv.StringOrNull(a.IntegrationStatus),
			"cloud_integration":   types.BoolValue(a.CloudIntegration),
		}
	},
}

func NewBitbucketInstallationsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(bitbucketInstallationsSpec)
}
