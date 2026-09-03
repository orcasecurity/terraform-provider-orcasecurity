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
		name             string
		hasIntegratePath bool
		existing         api_client.ScmUnitCommonFields
		adoptExisting    types.Bool
		wantBlock        bool
	}{
		// hasIntegratePath=true (GitLab/Azure DevOps/Bitbucket): existence alone blocks, since the
		// normal flow is a fresh Integrate — Get() finding anything means it predates this apply.
		{"has integrate path, empty unit, flag unset: block", true, api_client.ScmUnitCommonFields{}, types.BoolNull(), true},
		{"has integrate path, empty unit, flag true: allow", true, api_client.ScmUnitCommonFields{}, types.BoolValue(true), false},
		{"has integrate path, unit with repos, flag unset: block", true, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 3}, types.BoolNull(), true},
		{"has integrate path, unit with repos, flag true: allow", true, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 3}, types.BoolValue(true), false},

		// hasIntegratePath=false (GitHub): no fresh-create path exists, so Get() finds the unit on
		// every valid usage — existence carries no signal. Fall back to actual state at risk.
		{"no integrate path, empty unit, flag unset: allow", false, api_client.ScmUnitCommonFields{}, types.BoolNull(), false},
		{"no integrate path, empty unit, flag false: allow", false, api_client.ScmUnitCommonFields{}, types.BoolValue(false), false},
		{"no integrate path, empty unit, flag true: allow", false, api_client.ScmUnitCommonFields{}, types.BoolValue(true), false},
		{"no integrate path, has repos, flag unset: block", false, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 3}, types.BoolNull(), true},
		{"no integrate path, has repos, flag false: block", false, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 3}, types.BoolValue(false), true},
		{"no integrate path, has repos, flag true: allow", false, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 3}, types.BoolValue(true), false},
		{"no integrate path, one repo, flag unset: block", false, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 1}, types.BoolNull(), true},
		{"no integrate path, attached policies, flag unset: block", false, api_client.ScmUnitCommonFields{Policies: []api_client.ScmPolicyRef{{ID: "p1"}}}, types.BoolNull(), true},
		{"no integrate path, bound project, flag unset: block", false, api_client.ScmUnitCommonFields{Project: &api_client.ScmProjectRef{ID: "proj-1"}}, types.BoolNull(), true},
		{"no integrate path, bound project, flag true: allow", false, api_client.ScmUnitCommonFields{Project: &api_client.ScmProjectRef{ID: "proj-1"}}, types.BoolValue(true), false},

		// default_policies alone is never a signal: the API derives it as true whenever there are
		// no policies and no project (scm_schema.go's default_policies doc), for every SCM — so a
		// unit with nothing else attached must stay adoptable under the no-integrate-path fallback,
		// or every existing GitHub unit would need adopt_existing regardless of whether it's blank.
		{"no integrate path, default_policies=true, nothing else attached: allow", false, api_client.ScmUnitCommonFields{DefaultPolicies: true}, types.BoolNull(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guardAdopt(tt.hasIntegratePath, tt.existing, tt.adoptExisting); got != tt.wantBlock {
				t.Errorf("guardAdopt(%v, %+v, %v) = %v, want %v", tt.hasIntegratePath, tt.existing, tt.adoptExisting, got, tt.wantBlock)
			}
		})
	}
}

func TestAdoptGuardDetail(t *testing.T) {
	msg := adoptGuardDetail(`Account "acme" on installation "inst-1"`, api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 4})
	for _, want := range []string{"acme", "4 integrated repositories", "terraform import", "adopt_existing = true", "DE-INTEGRATE"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
	if one := adoptGuardDetail("unit", api_client.ScmUnitCommonFields{IntegratedRepositoriesCount: 1}); !strings.Contains(one, "1 integrated repository") {
		t.Errorf("singular repository not rendered: %s", one)
	}
	if withProject := adoptGuardDetail("unit", api_client.ScmUnitCommonFields{Project: &api_client.ScmProjectRef{ID: "proj-1"}}); !strings.Contains(withProject, "a bound project") {
		t.Errorf("project reason not rendered: %s", withProject)
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

	// Integrate succeeded but lookup cannot find an id to delete: must warn, not succeed silently.
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
