package shift_left_integration

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCreateUnitBody_DefaultsScanAllAndPolicies(t *testing.T) {
	ad := CreateUnitBody(
		types.StringValue("SCAN_ALL_INCLUDE_FUTURE"),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
	)
	if ad.InstallationMode != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Errorf("mode: %q", ad.InstallationMode)
	}
	if !ad.DefaultPolicies {
		t.Error("expected default_policies true when unset")
	}
	if ad.ConfigSettings.CommentsOnPullRequests != "ALWAYS" {
		t.Errorf("expected default comments ALWAYS, got %+v", ad.ConfigSettings)
	}
}

// PR scanning must be ENABLED by default on every provider, matching what the UI
// sends when integrating a new unit. This is a deliberate divergence from
// GitHub's backend default (disable_scan_pull_requests: true), which the provider
// only avoids because it always sends the field explicitly — so pin it.
func TestCreateUnitBody_EnablesPullRequestScanningByDefault(t *testing.T) {
	ad := CreateUnitBody(
		types.StringValue("SELECTED_REPOSITORIES"),
		types.BoolNull(),
		types.SetNull(types.StringType),
		nil,
		ProjectIntent{},
	)
	if ad.ConfigSettings.DisableScanPullRequests {
		t.Error("a new unit must default to disable_scan_pull_requests=false (PR scanning on)")
	}

	// An explicit opt-out must still win.
	optOut := CreateUnitBody(
		types.StringValue("SELECTED_REPOSITORIES"),
		types.BoolNull(),
		types.SetNull(types.StringType),
		&ConfigSettingsModel{PRSettingsModel: shift_left_common.PRSettingsModel{DisableScanPullRequests: types.BoolValue(true)}},
		ProjectIntent{},
	)
	if !optOut.ConfigSettings.DisableScanPullRequests {
		t.Error("an explicit disable_scan_pull_requests=true must be honoured")
	}
}
