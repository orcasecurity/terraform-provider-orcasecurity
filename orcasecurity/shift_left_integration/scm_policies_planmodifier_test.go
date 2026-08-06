package shift_left_integration

import (
	"testing"
)

func TestPoliciesIDsShouldClear(t *testing.T) {
	cases := []struct {
		name             string
		defaultTrue      bool
		projectID        string
		stateHasPolicies bool
		wantClear        bool
	}{
		{"default_policies clears existing", true, "", true, true},
		{"default_policies already empty", true, "", false, false},
		{"project_id clears existing", false, "proj-1", true, true},
		{"project_id already empty", false, "proj-1", false, false},
		{"neither keeps state", false, "", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policiesIDsShouldClear(tc.defaultTrue, tc.projectID, tc.stateHasPolicies)
			if got != tc.wantClear {
				t.Fatalf("got %v want %v", got, tc.wantClear)
			}
		})
	}
}
