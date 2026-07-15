package shift_left_project

import (
	"reflect"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func TestAttachedPolicyIDs(t *testing.T) {
	tests := []struct {
		name    string
		current *api_client.ShiftLeftProject
		want    []string
	}{
		{"no policies", &api_client.ShiftLeftProject{}, []string{}},
		{
			"preserves all attached policy ids",
			&api_client.ShiftLeftProject{Policies: []api_client.ShiftLeftProjectPolicy{
				{ID: "pol-1", Builtin: true},
				{ID: "pol-2", Builtin: false},
			}},
			[]string{"pol-1", "pol-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := attachedPolicyIDs(tt.current); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}
