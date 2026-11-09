package createrecovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

var errAmbiguous = fmt.Errorf("%w: 504", api_client.ErrAmbiguousWrite)

func ids(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, id := range v {
		m[id] = true
	}
	return m
}

func TestRecover_NonAmbiguousErrorPassesThrough(t *testing.T) {
	plain := errors.New("400 bad request")
	if _, _, err := Recover(context.Background(), plain, "x", ids(), nil); err != plain {
		t.Fatalf("err = %v, want the original error", err)
	}
}

func TestRecover_AdoptsSingleNewID(t *testing.T) {
	FastForTest(t)
	id, warning, err := Recover(context.Background(), errAmbiguous, "x", ids("old"), func() (map[string]bool, error) {
		return ids("old", "new"), nil
	})
	if err != nil || id != "new" || warning == "" {
		t.Fatalf("id=%q warning=%q err=%v", id, warning, err)
	}
}

func TestRecover_PollsUntilTheCreateShowsUp(t *testing.T) {
	FastForTest(t)
	calls := 0
	id, _, err := Recover(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
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

func TestRecover_NoSnapshotNeverAdopts(t *testing.T) {
	FastForTest(t)
	_, _, err := Recover(context.Background(), errAmbiguous, "x", nil, func() (map[string]bool, error) {
		return ids("new"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "import it") {
		t.Fatalf("err = %v", err)
	}
}

func TestRecover_SeveralNewIDsError(t *testing.T) {
	FastForTest(t)
	_, _, err := Recover(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
		return ids("a", "b"), nil
	})
	if err == nil || !strings.Contains(err.Error(), "2 new matches") || !strings.Contains(err.Error(), "a, b") {
		t.Fatalf("err = %v", err)
	}
}

func TestRecover_LookupFailureErrors(t *testing.T) {
	FastForTest(t)
	_, _, err := Recover(context.Background(), errAmbiguous, "x", ids(), func() (map[string]bool, error) {
		return nil, errors.New("list down")
	})
	if err == nil || !strings.Contains(err.Error(), "also failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestSnapshot_FailureDisablesAdoption(t *testing.T) {
	if got := Snapshot(func() (map[string]bool, error) { return nil, errors.New("down") }); got != nil {
		t.Fatalf("snapshot = %v, want nil", got)
	}
}
