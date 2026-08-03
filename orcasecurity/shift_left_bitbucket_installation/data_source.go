package shift_left_bitbucket_installation

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var installationsSpec = shift_left_common.ScmObjectListSpec[api_client.BitbucketInstallation]{
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

func NewInstallationsDataSource() datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(installationsSpec)
}

func installationsToListValue(rows []api_client.BitbucketInstallation) (types.List, diag.Diagnostics) {
	return installationsSpec.ListValue(rows)
}
