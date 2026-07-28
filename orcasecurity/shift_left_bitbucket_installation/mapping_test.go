package shift_left_bitbucket_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWriteBody_CarriesTokenDetails(t *testing.T) {
	body := writeBody(&resourceModel{
		Name:            types.StringValue("bb-conn"),
		ServerURL:       types.StringValue("https://bitbucket.org"),
		AccessToken:     types.StringValue("tok-123"),
		AccessTokenType: types.StringValue("PAT"),
		Username:        types.StringValue("alice"),
		AccountID:       types.StringValue("workspace-slug"),
	})
	if body.Name != "bb-conn" || body.ServerURL != "https://bitbucket.org" {
		t.Errorf("top-level fields: %+v", body)
	}
	if body.AccessTokenDetails == nil {
		t.Fatal("access_token_details must be present")
	}
	td := body.AccessTokenDetails
	if td.AccessToken != "tok-123" || td.AccessTokenType != "PAT" || td.Username != "alice" || td.AccountID != "workspace-slug" {
		t.Errorf("token details: %+v", td)
	}
}

func TestSetState_MapsNestedTokenDetails(t *testing.T) {
	m := &resourceModel{}
	setState(m, &api_client.BitbucketInstallation{
		ID:                "inst-1",
		Name:              "bb-conn",
		ServerURL:         "https://bitbucket.org",
		ExternalServerURL: "https://bitbucket.org/ext",
		IntegrationStatus: "ENABLED",
		CloudIntegration:  true,
		AccessTokenDetails: &api_client.BitbucketAccessTokenDetails{
			AccessTokenType: "TOKEN",
			Username:        "alice",
			AccountID:       "workspace-slug",
		},
	})
	if m.ID.ValueString() != "inst-1" || m.Name.ValueString() != "bb-conn" {
		t.Errorf("id/name: %+v", m)
	}
	if m.ServerURL.ValueString() != "https://bitbucket.org" || m.ExternalServerURL.ValueString() != "https://bitbucket.org/ext" {
		t.Errorf("urls: %+v", m)
	}
	if m.IntegrationStatus.ValueString() != "ENABLED" || !m.CloudIntegration.ValueBool() {
		t.Errorf("status/cloud: %+v", m)
	}
	if m.AccessTokenType.ValueString() != "TOKEN" || m.Username.ValueString() != "alice" || m.AccountID.ValueString() != "workspace-slug" {
		t.Errorf("token details not flattened: %+v", m)
	}
}

func TestSetState_NilTokenDetailsIsSafe(t *testing.T) {
	// A read that omits access_token_details must not panic and must yield empty
	// (non-null) strings for the token-derived attributes.
	m := &resourceModel{}
	setState(m, &api_client.BitbucketInstallation{ID: "inst-1", Name: "bb-conn"})
	if m.AccessTokenType.ValueString() != "" || m.Username.ValueString() != "" || m.AccountID.ValueString() != "" {
		t.Errorf("nil token details must map to empty strings: %+v", m)
	}
	if m.AccessTokenType.IsNull() {
		t.Error("token fields should be empty-value strings, not null")
	}
}
