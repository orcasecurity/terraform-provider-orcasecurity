package shift_left_integration

import (
	"errors"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakeUnit struct {
	ID   string
	Mode string
	Cfg  api_client.ShiftLeftConfigSettings
}

func TestWriteAdopted_NotFound(t *testing.T) {
	_, err := WriteAdopted(AdoptWriteRequest[fakeUnit]{
		Get: func() (*fakeUnit, error) { return nil, nil },
		Update: func(*fakeUnit, api_client.ScmInstallationUpdate) (*fakeUnit, error) {
			t.Fatal("Update must not be called")
			return nil, nil
		},
		Snapshot: func(u *fakeUnit) ExistingUnit {
			return ExistingUnit{InstallationMode: u.Mode, ConfigSettings: u.Cfg}
		},
	})
	if !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("expected ErrUnitNotFound, got %v", err)
	}
}

func TestWriteAdopted_AdoptsAndUpdates(t *testing.T) {
	var gotBody api_client.ScmInstallationUpdate
	out, err := WriteAdopted(AdoptWriteRequest[fakeUnit]{
		Get: func() (*fakeUnit, error) {
			return &fakeUnit{ID: "u1", Mode: "SELECTED_REPOSITORIES", Cfg: api_client.ShiftLeftConfigSettings{PrSummaryComment: "ALWAYS"}}, nil
		},
		Update: func(current *fakeUnit, body api_client.ScmInstallationUpdate) (*fakeUnit, error) {
			if current.ID != "u1" {
				t.Fatalf("Update must receive loaded unit, got %+v", current)
			}
			gotBody = body
			return &fakeUnit{ID: "u1", Mode: body.InstallationMode, Cfg: body.ConfigSettings}, nil
		},
		Snapshot: func(u *fakeUnit) ExistingUnit {
			return ExistingUnit{InstallationMode: u.Mode, ConfigSettings: u.Cfg}
		},
		PlanMode:    types.StringNull(),
		PlanDefault: types.BoolValue(false),
		PlanConfig: &ConfigSettingsModel{
			PRSettingsModel: PRSettingsModel{PrSummaryComment: types.StringValue("NEVER")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.ID != "u1" {
		t.Fatalf("bad out: %+v", out)
	}
	if gotBody.InstallationMode != "SELECTED_REPOSITORIES" {
		t.Errorf("expected mode hydrated from live, got %q", gotBody.InstallationMode)
	}
	if gotBody.ConfigSettings.PrSummaryComment != "NEVER" {
		t.Errorf("expected overlay pr_summary_comment NEVER, got %q", gotBody.ConfigSettings.PrSummaryComment)
	}
}

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
		&ConfigSettingsModel{PRSettingsModel: PRSettingsModel{DisableScanPullRequests: types.BoolValue(true)}},
		ProjectIntent{},
	)
	if !optOut.ConfigSettings.DisableScanPullRequests {
		t.Error("an explicit disable_scan_pull_requests=true must be honoured")
	}
}
