package business_unit

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strs(values ...string) []types.String {
	return stringSliceToTypes(values)
}

func assertJSONEqual(t *testing.T, got json.RawMessage, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("got invalid JSON %s: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("want invalid JSON %s: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("config mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestConfigFromModel(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name      string
		filter    *businessUnitFilterModel
		shiftLeft *businessUnitShiftLeftFilterModel
		want      string
	}{
		{
			name:   "cloud providers",
			filter: &businessUnitFilterModel{CloudProviders: strs("aws", "azure")},
			want:   `{"some":{"CloudProviders":["aws","azure"]}}`,
		},
		{
			name:   "cloud vendor ids",
			filter: &businessUnitFilterModel{CloudAccounts: strs("123")},
			want:   `{"some":{"CloudVendorIDs":["123"]}}`,
		},
		{
			name:   "deprecated cloud account ids",
			filter: &businessUnitFilterModel{CloudAccountIds: strs("123")},
			want:   `{"some":{"CloudVendorIDs":["123"]}}`,
		},
		{
			name:   "account tags",
			filter: &businessUnitFilterModel{AccountTags: strs("team|sec")},
			want:   `{"some":{"AccountTags":["team|sec"]}}`,
		},
		{
			name:   "cloud tags",
			filter: &businessUnitFilterModel{CloudTags: strs("env|Prod")},
			want:   `{"some":{"InventoryTags":["env|Prod"]}}`,
		},
		{
			name:   "custom tags",
			filter: &businessUnitFilterModel{CustomTags: strs("owner|me")},
			want:   `{"some":{"CustomTags":["owner|me"]}}`,
		},
		{
			name:      "shift left only",
			shiftLeft: &businessUnitShiftLeftFilterModel{ShiftLeftProjects: strs(projectID)},
			want:      `{"some":{"AppSecProjectIDs":["` + projectID + `"]}}`,
		},
		{
			name:      "cloud vendor ids and shift left are a union",
			filter:    &businessUnitFilterModel{CloudAccounts: strs("123")},
			shiftLeft: &businessUnitShiftLeftFilterModel{ShiftLeftProjects: strs(projectID)},
			want:      `{"or":[{"some":{"CloudVendorIDs":["123"]}},{"some":{"AppSecProjectIDs":["` + projectID + `"]}}]}`,
		},
		{
			name: "several filter fields are a union in fixed order",
			filter: &businessUnitFilterModel{
				CustomTags:     strs("owner|me"),
				CloudProviders: strs("gcp"),
			},
			want: `{"or":[{"some":{"CloudProviders":["gcp"]}},{"some":{"CustomTags":["owner|me"]}}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configFromModel(tt.filter, tt.shiftLeft)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, got, tt.want)
		})
	}
}

func TestConfigFromModel_Empty(t *testing.T) {
	for name, filter := range map[string]*businessUnitFilterModel{
		"nil":         nil,
		"empty lists": {CloudProviders: strs()},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := configFromModel(filter, &businessUnitShiftLeftFilterModel{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != nil {
				t.Errorf("expected nil config, got %s", got)
			}
		})
	}
}

func TestModelFromConfig_RoundTrip(t *testing.T) {
	projectID := "550e8400-e29b-41d4-a716-446655440000"
	tests := map[string]struct {
		filter    *businessUnitFilterModel
		shiftLeft *businessUnitShiftLeftFilterModel
	}{
		"cloud providers":  {filter: &businessUnitFilterModel{CloudProviders: strs("aws", "azure")}},
		"cloud vendor ids": {filter: &businessUnitFilterModel{CloudAccounts: strs("1", "2")}},
		"account tags":     {filter: &businessUnitFilterModel{AccountTags: strs("a|b")}},
		"cloud tags":       {filter: &businessUnitFilterModel{CloudTags: strs("Env|Prod")}},
		"custom tags":      {filter: &businessUnitFilterModel{CustomTags: strs("c|d")}},
		"shift left only":  {shiftLeft: &businessUnitShiftLeftFilterModel{ShiftLeftProjects: strs(projectID)}},
		"cloud vendor ids and shift left": {
			filter:    &businessUnitFilterModel{CloudAccounts: strs("1")},
			shiftLeft: &businessUnitShiftLeftFilterModel{ShiftLeftProjects: strs(projectID)},
		},
		"all filter fields": {filter: &businessUnitFilterModel{
			CloudProviders: strs("aws"),
			CloudAccounts:  strs("1"),
			AccountTags:    strs("a|b"),
			CloudTags:      strs("e|f"),
			CustomTags:     strs("c|d"),
		}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := configFromModel(tt.filter, tt.shiftLeft)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			filter, shiftLeft, ok := modelFromConfig(config)
			if !ok {
				t.Fatalf("expected %s to be representable", config)
			}
			if !reflect.DeepEqual(filter, tt.filter) {
				t.Errorf("filter mismatch: got %+v, want %+v", filter, tt.filter)
			}
			if !reflect.DeepEqual(shiftLeft, tt.shiftLeft) {
				t.Errorf("shift left mismatch: got %+v, want %+v", shiftLeft, tt.shiftLeft)
			}
		})
	}
}

func TestModelFromConfig_MergesRepeatedVariable(t *testing.T) {
	filter, shiftLeft, ok := modelFromConfig(json.RawMessage(
		`{"or":[{"some":{"CloudProviders":["aws"]}},{"some":{"CloudProviders":["gcp"]}}]}`,
	))
	if !ok {
		t.Fatal("expected representable")
	}
	if shiftLeft != nil {
		t.Errorf("expected no shift left filter, got %+v", shiftLeft)
	}
	if !reflect.DeepEqual(filter, &businessUnitFilterModel{CloudProviders: strs("aws", "gcp")}) {
		t.Errorf("unexpected filter: %+v", filter)
	}
}

func TestModelFromConfig_Empty(t *testing.T) {
	for _, config := range []string{"", "null", "{}"} {
		filter, shiftLeft, ok := modelFromConfig(json.RawMessage(config))
		if !ok || filter != nil || shiftLeft != nil {
			t.Errorf("config %q: got filter=%+v shiftLeft=%+v ok=%v", config, filter, shiftLeft, ok)
		}
	}
}

func TestModelFromConfig_NotRepresentable(t *testing.T) {
	for name, config := range map[string]string{
		"and":           `{"and":[{"some":{"CloudProviders":["aws"]}},{"all":{"CustomTags":["a|b"]}}]}`,
		"all leaf":      `{"all":{"CustomTags":["a|b"]}}`,
		"nested or":     `{"or":[{"or":[{"some":{"CloudProviders":["aws"]}}]}]}`,
		"hierarchy":     `{"some":{"HierarchyPath":[{"id":"r-1","name":"root","type":"root"}]}}`,
		"k8s namespace": `{"some":{"K8sNamespaces":["default"]}}`,
		"empty or":      `{"or":[]}`,
		"empty values":  `{"some":{"CloudProviders":[]}}`,
		"two vars":      `{"some":{"CloudProviders":["aws"],"CustomTags":["a|b"]}}`,
		"not an object": `["aws"]`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, ok := modelFromConfig(json.RawMessage(config)); ok {
				t.Errorf("expected %s to be not representable", config)
			}
		})
	}
}

func TestBusinessUnitRequest_FromFilterBlocks(t *testing.T) {
	plan := &businessUnitResourceModel{
		Name:   types.StringValue("bu"),
		Filter: &businessUnitFilterModel{CloudProviders: strs("aws")},
		Config: jsontypes.NewNormalizedNull(),
	}
	req, diags := businessUnitRequest(plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if req.BUType != "combined_filter" {
		t.Errorf("expected combined_filter, got %q", req.BUType)
	}
	assertJSONEqual(t, req.Config, `{"some":{"CloudProviders":["aws"]}}`)
}

func TestBusinessUnitRequest_FromConfigAttribute(t *testing.T) {
	config := `{"and":[{"some":{"CloudProviders":["aws"]}},{"all":{"CustomTags":["a|b"]}}]}`
	plan := &businessUnitResourceModel{
		Name:   types.StringValue("bu"),
		Config: jsontypes.NewNormalizedValue(config),
	}
	req, diags := businessUnitRequest(plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	assertJSONEqual(t, req.Config, config)
}

func TestBusinessUnitRequest_NoScope(t *testing.T) {
	plan := &businessUnitResourceModel{
		Name:   types.StringValue("bu"),
		Config: jsontypes.NewNormalizedNull(),
	}
	if _, diags := businessUnitRequest(plan); !diags.HasError() {
		t.Error("expected an error for a business unit without scope")
	}
}

func TestSetScopeInState_FilterBlocks(t *testing.T) {
	state := &businessUnitResourceModel{Config: jsontypes.NewNormalizedNull()}
	diags := setScopeInState(state, json.RawMessage(`{"some":{"CustomTags":["a|b"]}}`), false)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !reflect.DeepEqual(state.Filter, &businessUnitFilterModel{CustomTags: strs("a|b")}) {
		t.Errorf("unexpected filter: %+v", state.Filter)
	}
	if !state.Config.IsNull() {
		t.Errorf("expected config to stay null, got %s", state.Config.ValueString())
	}
}

func TestSetScopeInState_UnrepresentableWarnsAndDropsBlocks(t *testing.T) {
	state := &businessUnitResourceModel{
		Config: jsontypes.NewNormalizedNull(),
		Filter: &businessUnitFilterModel{CloudProviders: strs("aws")},
	}
	diags := setScopeInState(state, json.RawMessage(`{"all":{"CustomTags":["a|b"]}}`), false)
	if diags.WarningsCount() != 1 {
		t.Errorf("expected one warning, got %v", diags)
	}
	if state.Filter != nil || state.ShiftLeftFilter != nil || !state.Config.IsNull() {
		t.Errorf("expected empty scope so the next plan restores it, got %+v", state)
	}
}

func TestSetScopeInState_ImportKeepsUnrepresentableAsConfig(t *testing.T) {
	config := `{"all":{"CustomTags":["a|b"]}}`
	state := &businessUnitResourceModel{Config: jsontypes.NewNormalizedNull()}
	diags := setScopeInState(state, json.RawMessage(config), true)
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if state.Config.ValueString() != config {
		t.Errorf("expected config %s, got %s", config, state.Config.ValueString())
	}
}

func TestSetScopeInState_ConfigAttribute(t *testing.T) {
	config := `{"some":{"CloudProviders":["aws"]}}`
	state := &businessUnitResourceModel{Config: jsontypes.NewNormalizedValue(`{"some":{"CloudProviders":["gcp"]}}`)}
	diags := setScopeInState(state, json.RawMessage(config), false)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if state.Config.ValueString() != config || state.Filter != nil {
		t.Errorf("expected config %s and no filter blocks, got %+v", config, state)
	}
}
