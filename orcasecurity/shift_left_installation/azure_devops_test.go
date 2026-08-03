package shift_left_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAzureWriteBody_CarriesTokenAndAccount(t *testing.T) {
	body := azureWriteBody(&azureInstallationModel{
		Name:        types.StringValue("azure-conn"),
		ServerURL:   types.StringValue("https://dev.azure.com"),
		AccessToken: types.StringValue("pat-123"),
		AccountName: types.StringValue("my-org"),
	})
	if body.Name != "azure-conn" || body.ServerURL != "https://dev.azure.com" {
		t.Errorf("top-level fields: %+v", body)
	}
	if body.AccessTokenDetails == nil {
		t.Fatal("access_token_details must be present")
	}
	if body.AccessTokenDetails.AccessToken != "pat-123" || body.AccessTokenDetails.AccountName != "my-org" {
		t.Errorf("token details: %+v", body.AccessTokenDetails)
	}
}

func TestAzureSetState_MapsAPIFields(t *testing.T) {
	m := &azureInstallationModel{}
	azureSetState(m, &api_client.AzureDevopsInstallation{
		ID:                     "inst-1",
		Name:                   "azure-conn",
		ServerURL:              "https://dev.azure.com",
		AccessTokenType:        "SINGLE_ACCOUNT",
		AccessTokenAccountName: "my-org",
		ExternalServerURL:      "https://dev.azure.com/ext",
		IntegrationStatus:      "ENABLED",
		CloudIntegration:       true,
	})
	if m.ID.ValueString() != "inst-1" || m.Name.ValueString() != "azure-conn" {
		t.Errorf("id/name: %+v", m)
	}
	if m.ServerURL.ValueString() != "https://dev.azure.com" || m.ExternalServerURL.ValueString() != "https://dev.azure.com/ext" {
		t.Errorf("urls: %+v", m)
	}
	if m.AccessTokenType.ValueString() != "SINGLE_ACCOUNT" || m.AccessTokenAccountName.ValueString() != "my-org" {
		t.Errorf("token classification: %+v", m)
	}
	if m.IntegrationStatus.ValueString() != "ENABLED" || !m.CloudIntegration.ValueBool() {
		t.Errorf("status/cloud: %+v", m)
	}
	// The API never echoes the token, so setState must not touch access_token.
	if !m.AccessToken.IsNull() {
		t.Errorf("access_token must remain untouched by setState, got %#v", m.AccessToken)
	}
}

func TestAzureInstallationsToListValue(t *testing.T) {
	list, diags := azureInstallationsSpec.ListValue([]api_client.AzureDevopsInstallation{
		{ID: "inst-1", Name: "Azure", ServerURL: "https://dev.azure.com", CloudIntegration: true},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		t.Fatalf("expected one installation, got %#v", list)
	}
}
