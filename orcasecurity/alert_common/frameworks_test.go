package alert_common

import (
	"errors"
	"testing"
)

func TestReplaceFrameworks_NotReplacing_OnlyCallsUpdate(t *testing.T) {
	var clearCalled, updateCalled bool
	clearedButFailed, err := ReplaceFrameworks(false, true,
		func() error { clearCalled = true; return nil },
		func() error { updateCalled = true; return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clearCalled {
		t.Fatal("clear should not run when there were no old frameworks")
	}
	if !updateCalled {
		t.Fatal("update should run")
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false")
	}
}

func TestReplaceFrameworks_Replacing_ClearsThenUpdates(t *testing.T) {
	var order []string
	clearedButFailed, err := ReplaceFrameworks(true, true,
		func() error { order = append(order, "clear"); return nil },
		func() error { order = append(order, "update"); return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != "clear" || order[1] != "update" {
		t.Fatalf("expected clear then update, got %v", order)
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false on full success")
	}
}

func TestReplaceFrameworks_Replacing_ClearFails(t *testing.T) {
	wantErr := errors.New("boom")
	var updateCalled bool
	clearedButFailed, err := ReplaceFrameworks(true, true,
		func() error { return wantErr },
		func() error { updateCalled = true; return nil },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	if updateCalled {
		t.Fatal("update should not run when clear fails")
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false when clear itself failed")
	}
}

func TestReplaceFrameworks_Replacing_UpdateFailsAfterClear(t *testing.T) {
	wantErr := errors.New("boom")
	clearedButFailed, err := ReplaceFrameworks(true, true,
		func() error { return nil },
		func() error { return wantErr },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	if !clearedButFailed {
		t.Fatal("clearedButFailed should be true: clear succeeded remotely but update did not")
	}
}

func TestReplaceFrameworks_NotHadOld_NotHasNew_OnlyCallsUpdate(t *testing.T) {
	var clearCalled bool
	_, err := ReplaceFrameworks(false, false,
		func() error { clearCalled = true; return nil },
		func() error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clearCalled {
		t.Fatal("clear should not run when there was nothing to replace")
	}
}

func TestReplaceFrameworks_HadOld_NotHasNew_OnlyCallsUpdate(t *testing.T) {
	var clearCalled bool
	_, err := ReplaceFrameworks(true, false,
		func() error { clearCalled = true; return nil },
		func() error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clearCalled {
		t.Fatal("clear should not run when plan drops all frameworks (delete path handles this)")
	}
}
