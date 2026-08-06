package shift_left_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGitlabWriteBody_AlwaysSendsReadOnly(t *testing.T) {
	// read_only is always sent: the API resets an omitted read_only to false.
	body := gitlabWriteBody(&gitlabInstallationModel{
		Name:        types.StringValue("gl-conn"),
		ServerURL:   types.StringValue("https://gitlab.com"),
		AccessToken: types.StringValue("glpat-123"),
		ReadOnly:    types.BoolValue(true),
	})
	if body.Name != "gl-conn" || body.ServerURL != "https://gitlab.com" || body.AccessToken != "glpat-123" {
		t.Errorf("top-level fields: %+v", body)
	}
	if !body.ReadOnly {
		t.Errorf("read_only must be sent as true, got %+v", body)
	}

	// A null read_only in the model marshals as an explicit false (never omitted).
	def := gitlabWriteBody(&gitlabInstallationModel{Name: types.StringValue("gl-conn")})
	if def.ReadOnly {
		t.Errorf("null read_only must send false, got true")
	}
}

func TestGitlabSetState_MapsAPIFields(t *testing.T) {
	m := &gitlabInstallationModel{}
	gitlabSetState(m, &api_client.GitlabInstallation{
		ID:                "inst-1",
		Name:              "gl-conn",
		ServerURL:         "https://gitlab.com",
		ExternalServerURL: "https://gitlab.com/ext",
		AccessTokenName:   "orca-token",
		AccessTokenType:   "group",
		ReadOnly:          true,
		IntegrationStatus: "ENABLED",
		CloudIntegration:  true,
	})
	if m.ID.ValueString() != "inst-1" || m.Name.ValueString() != "gl-conn" {
		t.Errorf("id/name: %+v", m)
	}
	if m.ServerURL.ValueString() != "https://gitlab.com" || m.ExternalServerURL.ValueString() != "https://gitlab.com/ext" {
		t.Errorf("urls: %+v", m)
	}
	if m.AccessTokenName.ValueString() != "orca-token" || m.AccessTokenType.ValueString() != "group" {
		t.Errorf("token metadata: %+v", m)
	}
	if !m.ReadOnly.ValueBool() {
		t.Errorf("read_only: %+v", m.ReadOnly)
	}
	if m.IntegrationStatus.ValueString() != "ENABLED" || !m.CloudIntegration.ValueBool() {
		t.Errorf("status/cloud: %+v", m)
	}
	// The API never echoes the token, so setState must not touch access_token.
	if !m.AccessToken.IsNull() {
		t.Errorf("access_token must remain untouched by setState, got %#v", m.AccessToken)
	}
}

func TestGitlabInstallationsToListValue(t *testing.T) {
	list, diags := gitlabInstallationsSpec.ListValue([]api_client.GitlabInstallation{
		{ID: "inst-1", Name: "GitLab", ServerURL: "https://gitlab.com", ReadOnly: true, CloudIntegration: true},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		t.Fatalf("expected one installation, got %#v", list)
	}
}
