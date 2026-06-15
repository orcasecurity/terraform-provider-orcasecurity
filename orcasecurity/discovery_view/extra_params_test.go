package discovery_view

import (
	"reflect"
	"testing"
)

func TestBuildExtraParams_Empty(t *testing.T) {
	got := buildExtraParams(nil, "", nil)
	if len(got) != 0 {
		t.Errorf("expected empty extra_params, got %v", got)
	}
}

func TestBuildExtraParams_ColumnsOnly(t *testing.T) {
	got := buildExtraParams([]string{"OrcaScore", "CloudAccount"}, "", nil)

	columns, ok := got[extraParamsColumnsKey].(map[string]interface{})
	if !ok {
		t.Fatalf("expected %q to be an object, got %v", extraParamsColumnsKey, got[extraParamsColumnsKey])
	}
	if !reflect.DeepEqual(columns["keys"], []string{"OrcaScore", "CloudAccount"}) {
		t.Errorf("unexpected columns keys: %v", columns["keys"])
	}
	if _, exists := got[extraParamsSortKey]; exists {
		t.Error("did not expect sort2 to be set")
	}
	if _, exists := got[extraParamsGroupByKey]; exists {
		t.Error("did not expect groupBy2 to be set")
	}
}

func TestBuildExtraParams_All(t *testing.T) {
	got := buildExtraParams([]string{"OrcaScore"}, "-OrcaScore", []string{"AlertType"})

	if got[extraParamsSortKey] != "-OrcaScore" {
		t.Errorf("unexpected sort2: %v", got[extraParamsSortKey])
	}
	if !reflect.DeepEqual(got[extraParamsGroupByKey], []string{"AlertType"}) {
		t.Errorf("unexpected groupBy2: %v", got[extraParamsGroupByKey])
	}
	columns, ok := got[extraParamsColumnsKey].(map[string]interface{})
	if !ok || !reflect.DeepEqual(columns["keys"], []string{"OrcaScore"}) {
		t.Errorf("unexpected columns2: %v", got[extraParamsColumnsKey])
	}
}

func TestExtractColumns(t *testing.T) {
	testCases := []struct {
		name        string
		extraParams map[string]interface{}
		expected    []string
	}{
		{
			name: "valid columns2.keys (as returned by the API)",
			extraParams: map[string]interface{}{
				"columns2": map[string]interface{}{
					"hash":          "e40q6v",
					"keys":          []interface{}{"$overview", "CloudAccount", "OrcaScore"},
					"collapsedKeys": []interface{}{},
				},
			},
			expected: []string{"$overview", "CloudAccount", "OrcaScore"},
		},
		{name: "missing columns2", extraParams: map[string]interface{}{}, expected: nil},
		{name: "nil extra_params", extraParams: nil, expected: nil},
		{
			name:        "columns2 wrong type",
			extraParams: map[string]interface{}{"columns2": "not-an-object"},
			expected:    nil,
		},
		{
			name:        "keys wrong type",
			extraParams: map[string]interface{}{"columns2": map[string]interface{}{"keys": "nope"}},
			expected:    nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := extractColumns(testCase.extraParams)
			if !reflect.DeepEqual(got, testCase.expected) {
				t.Errorf("expected %v, got %v", testCase.expected, got)
			}
		})
	}
}

func TestExtractSort(t *testing.T) {
	testCases := []struct {
		name        string
		extraParams map[string]interface{}
		expected    string
	}{
		{name: "valid sort2", extraParams: map[string]interface{}{"sort2": "-OrcaScore"}, expected: "-OrcaScore"},
		{name: "missing sort2", extraParams: map[string]interface{}{}, expected: ""},
		{name: "sort2 wrong type", extraParams: map[string]interface{}{"sort2": 5}, expected: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := extractSort(testCase.extraParams); got != testCase.expected {
				t.Errorf("expected %q, got %q", testCase.expected, got)
			}
		})
	}
}

func TestExtractGroupBy(t *testing.T) {
	testCases := []struct {
		name        string
		extraParams map[string]interface{}
		expected    []string
	}{
		{
			name:        "valid groupBy2",
			extraParams: map[string]interface{}{"groupBy2": []interface{}{"AlertType", "CloudAccount"}},
			expected:    []string{"AlertType", "CloudAccount"},
		},
		{name: "empty groupBy2", extraParams: map[string]interface{}{"groupBy2": []interface{}{}}, expected: []string{}},
		{name: "missing groupBy2", extraParams: map[string]interface{}{}, expected: nil},
		{name: "groupBy2 wrong type", extraParams: map[string]interface{}{"groupBy2": "nope"}, expected: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := extractGroupBy(testCase.extraParams)
			if !reflect.DeepEqual(got, testCase.expected) {
				t.Errorf("expected %v, got %v", testCase.expected, got)
			}
		})
	}
}

// TestExtraParamsRoundTrip verifies that values built for the API can be read
// back into the same Terraform values (the create -> read consistency path).
func TestExtraParamsRoundTrip(t *testing.T) {
	columns := []string{"$overview", "CloudAccount", "OrcaScore"}
	sort := "-OrcaScore"
	groupBy := []string{"AlertType"}

	built := buildExtraParams(columns, sort, groupBy)

	// Simulate the API echoing the object back through JSON-like decoding,
	// where arrays come back as []interface{}.
	apiResponse := map[string]interface{}{
		extraParamsSortKey:    built[extraParamsSortKey],
		extraParamsGroupByKey: toInterfaceSlice(groupBy),
		extraParamsColumnsKey: map[string]interface{}{
			"keys": toInterfaceSlice(columns),
		},
	}

	if got := extractColumns(apiResponse); !reflect.DeepEqual(got, columns) {
		t.Errorf("columns round-trip mismatch: expected %v, got %v", columns, got)
	}
	if got := extractSort(apiResponse); got != sort {
		t.Errorf("sort round-trip mismatch: expected %q, got %q", sort, got)
	}
	if got := extractGroupBy(apiResponse); !reflect.DeepEqual(got, groupBy) {
		t.Errorf("group_by round-trip mismatch: expected %v, got %v", groupBy, got)
	}
}

func toInterfaceSlice(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
