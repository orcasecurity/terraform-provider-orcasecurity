package shift_left_installation

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeepOrSet(t *testing.T) {
	cases := []struct {
		name  string
		prior types.String
		api   string
		want  types.String
	}{
		{"api value wins over unknown", types.StringUnknown(), "PAT", types.StringValue("PAT")},
		{"api value wins over prior", types.StringValue("TOKEN"), "PAT", types.StringValue("PAT")},
		{"silent api resolves unknown to null", types.StringUnknown(), "", types.StringNull()},
		{"silent api keeps prior value", types.StringValue("alice"), "", types.StringValue("alice")},
		{"silent api keeps null", types.StringNull(), "", types.StringNull()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := c.prior
			keepOrSet(&v, c.api)
			if !v.Equal(c.want) {
				t.Errorf("keepOrSet(%v, %q) = %v, want %v", c.prior, c.api, v, c.want)
			}
		})
	}
}
