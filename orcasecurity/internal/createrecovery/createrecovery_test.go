package createrecovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/backoff"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

var errAmbiguous = fmt.Errorf("%w: 504", api_client.ErrAmbiguousWrite)

func ids(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, id := range v {
		m[id] = true
	}
	return m
}

func TestRecoverID_NonAmbiguousErrorPassesThrough(t *testing.T) {
	plain := errors.New("400 bad request")
	if _, _, err := recoverID(context.Background(), plain, "x", ids(), nil); err != plain {
		t.Fatalf("err = %v, want the original error", err)
	}
}

func TestRecoverID_AdoptsSingleNewID(t *testing.T) {
	backoff.FastForTest(t)
	id, warning, err := recoverID(context.Background(), errAmbiguous, "x", ids("old"), func() (map[string]bool, error) {
		return ids("old", "new"), nil
	})
	if err != nil || id != "new" || warning == "" {
		t.Fatalf("id=%q warning=%q err=%v", id, warning, err)
	}
}

func TestRecoverID_PollsUntilTheCreateShowsUp(t *testing.T) {
	backoff.FastForTest(t)
	calls := 0
	id, _, err := recoverID(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
		calls++
		if calls < 3 {
			return ids(), nil
		}
		return ids("new"), nil
	})
	if err != nil || id != "new" || calls != 3 {
		t.Fatalf("id=%q calls=%d err=%v", id, calls, err)
	}
}

func TestRecoverID_NoSnapshotNeverAdopts(t *testing.T) {
	backoff.FastForTest(t)
	_, _, err := recoverID(context.Background(), errAmbiguous, "x", nil, func() (map[string]bool, error) {
		return ids("new"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "import it") {
		t.Fatalf("err = %v", err)
	}
}

func TestRecoverID_SeveralNewIDsError(t *testing.T) {
	backoff.FastForTest(t)
	_, _, err := recoverID(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
		return ids("a", "b"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "2 new matches") || !strings.Contains(err.Error(), "a, b") {
		t.Fatalf("err = %v", err)
	}
}

func TestRecoverID_LookupFailureErrors(t *testing.T) {
	backoff.FastForTest(t)
	_, _, err := recoverID(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
		return nil, errors.New("list down")
	})
	if err == nil || !strings.Contains(err.Error(), "also failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestSnapshot_FailureDisablesAdoption(t *testing.T) {
	if got := snapshot(context.Background(), "x", func() (map[string]bool, error) { return nil, errors.New("down") }); got != nil {
		t.Fatalf("snapshot = %v, want nil", got)
	}
}

func TestRun_CreateWithoutAnIDIsAnError(t *testing.T) {
	type thing struct{ ID string }
	for name, created := range map[string]*thing{"nil": nil, "empty id": {}} {
		gets := 0
		op := api_client.CreateOp[thing]{
			What:   `thing named "x"`,
			Create: func() (*thing, error) { return created, nil },
			ID:     func(t *thing) string { return t.ID },
			IDs:    func() (map[string]bool, error) { return ids(), nil },
			Get: func(string) (*thing, error) {
				gets++
				return nil, nil
			},
		}
		var diags diag.Diagnostics
		got, err := Run(context.Background(), op, &diags)
		if got != nil || err == nil || !strings.Contains(err.Error(), `thing named "x"`) || !strings.Contains(err.Error(), "returned no object id") {
			t.Fatalf("%s: got=%v err=%v, want a returned-no-object-id error", name, got, err)
		}
		if gets != 0 || len(diags) != 0 {
			t.Errorf("%s: gets=%d diags=%v, want no read-back and no diagnostics", name, gets, diags)
		}
	}
}
