package shift_left_integration

import (
	"errors"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakeUnit struct {
	Mode     string
	Policies []api_client.ScmPolicyRef
	Project  *api_client.ScmProjectRef
	Cfg      api_client.ShiftLeftConfigSettings
}

func TestWriteAdopted_NotFound(t *testing.T) {
	_, err := WriteAdopted(AdoptWriteRequest[fakeUnit]{
		Get: func() (*fakeUnit, error) { return nil, nil },
		Update: func(api_client.ScmInstallationUpdate) (*fakeUnit, error) {
			t.Fatal("update should not be called")
			return nil, nil
		},
		Snapshot:     func(*fakeUnit) ExistingUnit { return ExistingUnit{} },
		PlanMode:     types.StringNull(),
		PlanDefault:  types.BoolNull(),
		PlanPolicies: types.SetNull(types.StringType),
	})
	if !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("expected ErrUnitNotFound, got %v", err)
	}
}

func TestWriteAdopted_AdoptsAndUpdates(t *testing.T) {
	var gotBody api_client.ScmInstallationUpdate
	out, err := WriteAdopted(AdoptWriteRequest[fakeUnit]{
		Get: func() (*fakeUnit, error) {
			return &fakeUnit{
				Mode:     "SCAN_ALL_INCLUDE_FUTURE",
				Policies: []api_client.ScmPolicyRef{{ID: "pol-1"}},
				Cfg:      api_client.ShiftLeftConfigSettings{PrSummaryComment: "ALWAYS"},
			}, nil
		},
		Update: func(body api_client.ScmInstallationUpdate) (*fakeUnit, error) {
			gotBody = body
			return &fakeUnit{Mode: body.InstallationMode}, nil
		},
		Snapshot: func(u *fakeUnit) ExistingUnit {
			return ExistingFromAPI(u.Mode, false, u.Policies, u.Project, u.Cfg)
		},
		PlanMode:     types.StringNull(),
		PlanDefault:  types.BoolValue(false),
		PlanPolicies: types.SetNull(types.StringType),
		PlanConfig:   &ConfigSettingsModel{PrSummaryComment: types.StringValue("NEVER")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.Mode != "SCAN_ALL_INCLUDE_FUTURE" {
		t.Fatalf("unexpected result: %+v", out)
	}
	if len(gotBody.Policies) != 1 || gotBody.Policies[0] != "pol-1" {
		t.Fatalf("expected existing policies preserved, got %v", gotBody.Policies)
	}
	if gotBody.ConfigSettings.PrSummaryComment != "NEVER" {
		t.Fatalf("expected overlay applied, got %+v", gotBody.ConfigSettings)
	}
}

func TestPolicyIDsFromRefs(t *testing.T) {
	got := PolicyIDsFromRefs([]api_client.ScmPolicyRef{{ID: "a"}, {ID: "b"}})
	elems := got.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 ids, got %v", elems)
	}
	_ = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("a"), types.StringValue("b")})
}

func TestExpandUpdate_DefaultPoliciesClearsIds(t *testing.T) {
	body := ExpandUpdate(
		types.StringNull(),
		types.BoolValue(true),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1")}),
		nil,
	)
	if len(body.Policies) != 0 {
		t.Errorf("expected empty policies when default_policies=true, got %v", body.Policies)
	}
	if !body.DefaultPolicies {
		t.Error("default_policies should be true")
	}
}

func TestExpandUpdate_ExplicitPolicies(t *testing.T) {
	body := ExpandUpdate(
		types.StringNull(),
		types.BoolValue(false),
		types.SetValueMust(types.StringType, []attr.Value{types.StringValue("pol-1"), types.StringValue("pol-2")}),
		nil,
	)
	if len(body.Policies) != 2 {
		t.Errorf("expected 2 policies, got %v", body.Policies)
	}
}
