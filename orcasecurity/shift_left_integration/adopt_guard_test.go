package shift_left_integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

type rollbackTestUnit struct{ api_client.ScmUnitCommonFields }
type rollbackTestModel struct{}

func rollbackTestOps(del func(*rollbackTestModel) error) AdoptedUnitOps[rollbackTestUnit, rollbackTestModel] {
	return AdoptedUnitOps[rollbackTestUnit, rollbackTestModel]{
		Describe: func(*rollbackTestModel) string { return `GitLab group "acme"` },
		Delete:   del,
	}
}

func assertExactlyOneWarning(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.WarningsCount() != 1 {
		t.Fatalf("expected exactly one warning, got %v", diags)
	}
	if diags.HasError() {
		t.Fatalf("expected exactly one warning, got %v", diags)
	}
}

func assertWarningDetailContains(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	if detail := diags.Warnings()[0].Detail(); !strings.Contains(detail, want) {
		t.Errorf("warning detail missing %q: %s", want, detail)
	}
}

func assertWarningSummaryContains(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	if summary := diags.Warnings()[0].Summary(); !strings.Contains(summary, want) {
		t.Errorf("warning summary missing %q: %s", want, summary)
	}
}

func rollbackLookupMissDelete(delCalls *int) func(*rollbackTestModel) error {
	return func(*rollbackTestModel) error {
		return DeleteByLookup(
			"",
			func() (*struct{ ID string }, error) { return nil, nil },
			func(u *struct{ ID string }) string { return u.ID },
			func(string) error { *delCalls++; return nil },
		)
	}
}

// Create integrates and then reads the unit back. A failed read-back leaves an integration that
// Terraform never records, so it has to be undone (or reported when it cannot be).
func TestRollbackIntegration(t *testing.T) {
	t.Run("de-integrates the unit it just created", func(t *testing.T) {
		deletes := 0
		var diags diag.Diagnostics
		rollbackTestOps(func(*rollbackTestModel) error { deletes++; return nil }).
			rollbackIntegration(context.Background(), &rollbackTestModel{}, &diags)
		if deletes != 1 {
			t.Errorf("expected one de-integrate call, got %d", deletes)
		}
		if len(diags) != 0 {
			t.Errorf("a clean rollback needs no diagnostics, got %v", diags)
		}
	})

	t.Run("warns when the rollback itself fails", func(t *testing.T) {
		var diags diag.Diagnostics
		rollbackTestOps(func(*rollbackTestModel) error { return errors.New("delete rejected") }).
			rollbackIntegration(context.Background(), &rollbackTestModel{}, &diags)
		assertExactlyOneWarning(t, diags)
		assertWarningDetailContains(t, diags, "delete rejected")
	})

	t.Run("warns when the resource cannot de-integrate at all", func(t *testing.T) {
		var diags diag.Diagnostics
		rollbackTestOps(nil).rollbackIntegration(context.Background(), &rollbackTestModel{}, &diags)
		assertExactlyOneWarning(t, diags)
		assertWarningSummaryContains(t, diags, `GitLab group "acme"`)
	})

	// The create path that matters: Integrate succeeded, Get returned (nil, nil),
	// DeleteByLookup re-runs the same empty-id lookup and used to return success
	// with no warning — the silent orphan.
	t.Run("warns when DeleteByLookup finds nothing to tear down", func(t *testing.T) {
		var diags diag.Diagnostics
		delCalls := 0
		rollbackTestOps(rollbackLookupMissDelete(&delCalls)).
			rollbackIntegration(context.Background(), &rollbackTestModel{}, &diags)
		if delCalls != 0 {
			t.Errorf("DELETE must not run when lookup misses, got %d calls", delCalls)
		}
		assertExactlyOneWarning(t, diags)
		assertWarningDetailContains(t, diags, "could not be found again")
	})
}
