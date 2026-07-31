package shift_left_gitlab_installation

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func TestInstallationsToListValue(t *testing.T) {
	list, diags := installationsToListValue([]api_client.GitlabInstallation{
		{ID: "inst-1", Name: "GitLab", ServerURL: "https://gitlab.com", ReadOnly: true, CloudIntegration: true},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		t.Fatalf("expected one installation, got %#v", list)
	}
}
