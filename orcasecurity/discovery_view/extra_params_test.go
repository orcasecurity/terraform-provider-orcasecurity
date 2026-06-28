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
	got := buildExtraParams(
		[]string{"OrcaScore"},
		"-OrcaScore",
		[]groupByAPIEntry{{Key: "AlertType"}},
	)

	if got[extraParamsSortKey] != "-OrcaScore" {
		t.Errorf("unexpected sort2: %v", got[extraParamsSortKey])
	}
	expectedGroupBy := []map[string]interface{}{{"key": "AlertType"}}
	if !reflect.DeepEqual(got[extraParamsGroupByKey], expectedGroupBy) {
		t.Errorf("unexpected groupBy2: %v", got[extraParamsGroupByKey])
	}
	columns, ok := got[extraParamsColumnsKey].(map[string]interface{})
	if !ok || !reflect.DeepEqual(columns["keys"], []string{"OrcaScore"}) {
		t.Errorf("unexpected columns2: %v", got[extraParamsColumnsKey])
	}
}

func TestBuildExtraParams_GroupByWithSort(t *testing.T) {
	got := buildExtraParams(
		nil,
		"",
		[]groupByAPIEntry{
			{
				Key: "CloudAccount.Name",
				Sort: []groupBySortAPIEntry{
					{Field: "COUNT", Direction: "desc"},
				},
			},
		},
	)

	expected := []map[string]interface{}{
		{
			"key": "CloudAccount.Name",
			"sort": []map[string]interface{}{
				{"field": "COUNT", "direction": "desc"},
			},
		},
	}
	if !reflect.DeepEqual(got[extraParamsGroupByKey], expected) {
		t.Errorf("unexpected groupBy2: %v", got[extraParamsGroupByKey])
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
		expected    []groupByAPIEntry
	}{
		{
			name:        "legacy string entries",
			extraParams: map[string]interface{}{"groupBy2": []interface{}{"AlertType", "CloudAccount"}},
			expected:    []groupByAPIEntry{{Key: "AlertType"}, {Key: "CloudAccount"}},
		},
		{
			name: "object entries with sort",
			extraParams: map[string]interface{}{
				"groupBy2": []interface{}{
					map[string]interface{}{
						"key": "CloudAccount.Name",
						"sort": []interface{}{
							map[string]interface{}{"field": "COUNT", "direction": "desc"},
						},
					},
				},
			},
			expected: []groupByAPIEntry{
				{
					Key:  "CloudAccount.Name",
					Sort: []groupBySortAPIEntry{{Field: "COUNT", Direction: "desc"}},
				},
			},
		},
		{
			name: "object entries without sort",
			extraParams: map[string]interface{}{
				"groupBy2": []interface{}{
					map[string]interface{}{"key": "AlertType"},
				},
			},
			expected: []groupByAPIEntry{{Key: "AlertType"}},
		},
		{name: "empty groupBy2", extraParams: map[string]interface{}{"groupBy2": []interface{}{}}, expected: []groupByAPIEntry{}},
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
	groupBy := []groupByAPIEntry{
		{
			Key: "CloudAccount.Name",
			Sort: []groupBySortAPIEntry{
				{Field: "COUNT", Direction: "desc"},
			},
		},
	}

	built := buildExtraParams(columns, sort, groupBy)

	// Simulate the API echoing the object back through JSON-like decoding,
	// where arrays come back as []interface{} and objects as map[string]interface{}.
	apiResponse := map[string]interface{}{
		extraParamsSortKey:    built[extraParamsSortKey],
		extraParamsGroupByKey: groupByToInterfaceSlice(groupBy),
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

func groupByToInterfaceSlice(entries []groupByAPIEntry) []interface{} {
	out := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		obj := map[string]interface{}{"key": entry.Key}
		if len(entry.Sort) > 0 {
			sortItems := make([]interface{}, 0, len(entry.Sort))
			for _, s := range entry.Sort {
				sortItems = append(sortItems, map[string]interface{}{
					"field":     s.Field,
					"direction": s.Direction,
				})
			}
			obj["sort"] = sortItems
		}
		out = append(out, obj)
	}
	return out
}

func toInterfaceSlice(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
