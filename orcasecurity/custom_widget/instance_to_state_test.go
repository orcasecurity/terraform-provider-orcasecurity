package custom_widget

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

const (
	testDonutWidgetName   = "My Donut"
	errInstanceToStateFmt = "instanceToState: %v"
)

func TestApiWidgetTypeToTerraform(t *testing.T) {
	tests := []struct {
		apiType string
		want    string
	}{
		{"PIE_CHART_SINGLE", "donut"},
		{"ASSETS_TABLE", "asset-table"},
		{"ALERTS_TABLE", "alert-table"},
		{"donut", "donut"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := apiWidgetTypeToTerraform(tt.apiType)
		if got != tt.want {
			t.Errorf("apiWidgetTypeToTerraform(%q) = %q, want %q", tt.apiType, got, tt.want)
		}
	}
}

func TestInstanceToStateDonutWidget(t *testing.T) {
	instance := &api_client.CustomWidget{
		ID:                "widget-123",
		Name:              testDonutWidgetName,
		OrganizationLevel: true,
		ViewType:          "customs_widgets",
		ExtraParameters: api_client.CustomWidgetExtraParameters{
			Type:              "PIE_CHART_SINGLE",
			Category:          "Custom",
			EmptyStateMessage: "No data",
			Size:              "sm",
			IsNew:             true,
			Title:             testDonutWidgetName,
			Subtitle:          "Sub",
			Description:       "Desc",
			Settings: []api_client.CustomWidgetExtraParametersSettings{
				{
					Size:    "sm",
					Columns: []string{"col1"},
					Field: api_client.CustomWidgetExtraParametersSettingsField{
						Name: "Region",
						Type: "str",
					},
					RequestParameters: api_client.RequestParams{
						Query:            map[string]interface{}{"type": "object_set"},
						GroupBy:          []string{"Type"},
						GroupByList:      []string{"CloudAccount.Name"},
						Limit:            10,
						StartAtIndex:     0,
						EnablePagination: true,
						OrderBy:          []string{"-Score"},
					},
				},
			},
		},
	}

	state, err := instanceToState(instance)
	if err != nil {
		t.Fatalf(errInstanceToStateFmt, err)
	}

	if state.ID.ValueString() != "widget-123" {
		t.Errorf("ID = %q, want widget-123", state.ID.ValueString())
	}
	if state.Name.ValueString() != testDonutWidgetName {
		t.Errorf("Name = %q, want %q", state.Name.ValueString(), testDonutWidgetName)
	}
	if !state.OrganizationLevel.ValueBool() {
		t.Error("OrganizationLevel = false, want true")
	}
	if state.ExtraParameters == nil {
		t.Fatal("ExtraParameters is nil")
	}
	if state.ExtraParameters.Type.ValueString() != "donut" {
		t.Errorf("ExtraParameters.Type = %q, want donut", state.ExtraParameters.Type.ValueString())
	}
	if state.ExtraParameters.Settings.Field == nil {
		t.Fatal("Settings.Field is nil, expected field with Region/str")
	}
	if state.ExtraParameters.Settings.Field.Name.ValueString() != "Region" {
		t.Errorf("Field.Name = %q, want Region", state.ExtraParameters.Settings.Field.Name.ValueString())
	}
}

func TestInstanceToStateEmptySettings(t *testing.T) {
	instance := &api_client.CustomWidget{
		ID:                "widget-empty",
		Name:              "Empty Widget",
		OrganizationLevel: false,
		ViewType:          "customs_widgets",
		ExtraParameters: api_client.CustomWidgetExtraParameters{
			Type:              "ASSETS_TABLE",
			Category:          "Custom",
			EmptyStateMessage: "",
			Size:              "md",
			IsNew:             false,
			Title:             "Empty",
			Subtitle:          "",
			Description:       "",
			Settings:          []api_client.CustomWidgetExtraParametersSettings{},
		},
	}

	state, err := instanceToState(instance)
	if err != nil {
		t.Fatalf(errInstanceToStateFmt, err)
	}

	if state.ExtraParameters.Type.ValueString() != "asset-table" {
		t.Errorf("Type = %q, want asset-table", state.ExtraParameters.Type.ValueString())
	}
	if state.ExtraParameters.Settings.Field != nil {
		t.Error("Settings.Field should be nil for empty settings")
	}
}

func TestInstanceToStateAlertTable(t *testing.T) {
	instance := &api_client.CustomWidget{
		ID:   "alert-widget",
		Name: "Alerts",
		ExtraParameters: api_client.CustomWidgetExtraParameters{
			Type: "ALERTS_TABLE",
			Settings: []api_client.CustomWidgetExtraParametersSettings{
				{
					Columns: []string{"alert", "status"},
					RequestParameters: api_client.RequestParams{
						GroupBy: []string{"Name"},
					},
				},
			},
		},
	}

	state, err := instanceToState(instance)
	if err != nil {
		t.Fatalf(errInstanceToStateFmt, err)
	}

	if state.ExtraParameters.Type.ValueString() != "alert-table" {
		t.Errorf("Type = %q, want alert-table", state.ExtraParameters.Type.ValueString())
	}
}

func TestApiSettingsToStateSettingsEmptyField(t *testing.T) {
	s := api_client.CustomWidgetExtraParametersSettings{
		Field: api_client.CustomWidgetExtraParametersSettingsField{},
		RequestParameters: api_client.RequestParams{
			GroupBy: []string{"Type"},
		},
	}

	settings, err := apiSettingsToStateSettings(s)
	if err != nil {
		t.Fatalf("apiSettingsToStateSettings: %v", err)
	}

	if settings.Field != nil {
		t.Error("Field should be nil when Name and Type are empty")
	}
}

func TestInstanceToStateV2RequestParams2(t *testing.T) {
	// V2 API returns requestParams2 in settings; provider should parse it
	instance := &api_client.CustomWidget{
		ID:   "v2-widget",
		Name: "All Cloud Assets",
		ExtraParameters: api_client.CustomWidgetExtraParameters{
			Type: "PIE_CHART_SINGLE",
			Settings: []api_client.CustomWidgetExtraParametersSettings{
				{
					Size:  "sm",
					Field: api_client.CustomWidgetExtraParametersSettingsField{Name: "Type", Type: "str"},
					RequestParams2: &api_client.RequestParams{
						Query:       map[string]interface{}{"models": []interface{}{"Inventory"}, "type": "object_set"},
						GroupBy:     []string{"Type"},
						GroupByList: []string{"Type"},
					},
				},
			},
		},
	}

	state, err := instanceToState(instance)
	if err != nil {
		t.Fatalf(errInstanceToStateFmt, err)
	}

	if state.ID.ValueString() != "v2-widget" {
		t.Errorf("ID = %q, want v2-widget", state.ID.ValueString())
	}
	if state.ExtraParameters.Type.ValueString() != "donut" {
		t.Errorf("Type = %q, want donut (PIE_CHART_SINGLE)", state.ExtraParameters.Type.ValueString())
	}
	if state.ExtraParameters.Settings.RequestParameters.Query.ValueString() == "" {
		t.Error("Query should be populated from requestParams2")
	}
	if len(state.ExtraParameters.Settings.RequestParameters.GroupBy) != 1 || state.ExtraParameters.Settings.RequestParameters.GroupBy[0].ValueString() != "Type" {
		t.Errorf("GroupBy = %v, want [Type]", state.ExtraParameters.Settings.RequestParameters.GroupBy)
	}
}

func TestStringSliceToTypesStrings(t *testing.T) {
	got := stringSliceToTypesStrings([]string{"a", "b", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].ValueString() != "a" || got[1].ValueString() != "b" || got[2].ValueString() != "c" {
		t.Errorf("got %v", got)
	}
}

func TestColumnsFromAPI(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := columnsFromAPI(nil)
		if !got.IsNull() {
			t.Error("expected null for nil columns")
		}
	})
	t.Run("with values", func(t *testing.T) {
		got := columnsFromAPI([]string{"col1", "col2"})
		if got.IsNull() {
			t.Error("expected non-null for non-empty columns")
		}
	})
}

func TestOrderByFromAPI(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := orderByFromAPI([]string{})
		if !got.IsNull() {
			t.Error("expected null for empty orderBy")
		}
	})
	t.Run("with values", func(t *testing.T) {
		got := orderByFromAPI([]string{"-Score"})
		if got.IsNull() {
			t.Error("expected non-null for non-empty orderBy")
		}
	})
}
