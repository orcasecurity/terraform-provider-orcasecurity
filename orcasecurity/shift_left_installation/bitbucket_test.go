package shift_left_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBitbucketWriteBody_CarriesTokenDetails(t *testing.T) {
	body := bitbucketWriteBody(&bitbucketInstallationModel{
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

func TestBitbucketSetState_MapsNestedTokenDetails(t *testing.T) {
	m := &bitbucketInstallationModel{}
	bitbucketSetState(m, &api_client.BitbucketInstallation{
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

func TestBitbucketSetState_NilTokenDetailsLeavesEchoFieldsUntouched(t *testing.T) {
	// A read that omits access_token_details must not wipe configured echo fields
	// (same rule as access_token: the API is silent, so prior state wins).
	m := &bitbucketInstallationModel{
		AccessTokenType: types.StringValue("PAT"),
		Username:        types.StringValue("alice"),
		AccountID:       types.StringValue("ws"),
	}
	bitbucketSetState(m, &api_client.BitbucketInstallation{ID: "inst-1", Name: "bb-conn"})
	if m.AccessTokenType.ValueString() != "PAT" || m.Username.ValueString() != "alice" || m.AccountID.ValueString() != "ws" {
		t.Errorf("nil token details must leave echo fields untouched: %+v", m)
	}
}

func TestBitbucketSetState_EmptyEchoFieldsLeavePriorUntouched(t *testing.T) {
	m := &bitbucketInstallationModel{Username: types.StringValue("alice")}
	bitbucketSetState(m, &api_client.BitbucketInstallation{
		ID:   "inst-1",
		Name: "bb-conn",
		AccessTokenDetails: &api_client.BitbucketAccessTokenDetails{
			AccessTokenType: "PAT",
			Username:        "", // API null → ""
			AccountID:       "",
		},
	})
	if m.AccessTokenType.ValueString() != "PAT" {
		t.Errorf("non-empty type should update: %v", m.AccessTokenType)
	}
	if m.Username.ValueString() != "alice" {
		t.Errorf("empty username must not wipe configured value: %v", m.Username)
	}
}

func TestBitbucketInstallationsToListValue(t *testing.T) {
	list, diags := bitbucketInstallationsSpec.ListValue([]api_client.BitbucketInstallation{
		{
			ID:   "inst-1",
			Name: "Bitbucket",
			AccessTokenDetails: &api_client.BitbucketAccessTokenDetails{
				AccessTokenType: "TOKEN",
				AccountID:       "my-workspace",
			},
			CloudIntegration: true,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		t.Fatalf("expected one installation, got %#v", list)
	}
}
