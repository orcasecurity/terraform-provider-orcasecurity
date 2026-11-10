package api_client

import (
	"encoding/json"
	"testing"
)

func TestShiftLeftCveExceptionListRead_DisabledIsNilWhenOmitted(t *testing.T) {
	for body, want := range map[string]*bool{
		`{"id":"l1","name":"n","vulnerabilities":[{"cve_id":"c","disabled":true}]}`: nil,
		`{"id":"l1","name":"n","disabled":false}`:                                   new(bool),
	} {
		var got ShiftLeftCveExceptionListRead
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatal(err)
		}
		if (got.Disabled == nil) != (want == nil) || (want != nil && *got.Disabled != *want) || got.ID != "l1" {
			t.Errorf("%s: disabled=%v id=%q, want disabled=%v", body, got.Disabled, got.ID, want)
		}
	}
}
