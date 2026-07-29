package shift_left_integration

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGuardAdopt(t *testing.T) {
	tests := []struct {
		name          string
		repoCount     int64
		adoptExisting types.Bool
		wantBlock     bool
	}{
		{"empty unit, flag unset: allow", 0, types.BoolNull(), false},
		{"empty unit, flag false: allow", 0, types.BoolValue(false), false},
		{"has repos, flag unset: block", 3, types.BoolNull(), true},
		{"has repos, flag false: block", 3, types.BoolValue(false), true},
		{"has repos, flag true: allow", 3, types.BoolValue(true), false},
		{"one repo, flag unset: block", 1, types.BoolNull(), true},
		{"empty unit, flag true: allow", 0, types.BoolValue(true), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardAdopt(tt.repoCount, tt.adoptExisting); got != tt.wantBlock {
				t.Errorf("guardAdopt(%d, %v) = %v, want %v", tt.repoCount, tt.adoptExisting, got, tt.wantBlock)
			}
		})
	}
}

func TestAdoptGuardDetail(t *testing.T) {
	msg := adoptGuardDetail(`Account "acme" on installation "inst-1"`, 4)
	for _, want := range []string{"acme", "4 integrated repositories", "terraform import", "adopt_existing = true", "DE-INTEGRATE"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
	if one := adoptGuardDetail("unit", 1); !strings.Contains(one, "1 integrated repository ") {
		t.Errorf("singular repository not rendered: %s", one)
	}
}
