package alert_common

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// request stands in for an alert update payload: the helper only has to be able
// to copy it and blank its links.
type request struct {
	Name       string
	Frameworks []string
}

func clearLinks(r *request) { r.Frameworks = nil }

func withLinks(t *testing.T) types.List {
	t.Helper()
	list, diags := FrameworksToList(context.Background(), []Framework{{
		Name:     types.StringValue("framework"),
		Section:  types.StringValue("section"),
		Priority: types.StringValue("medium"),
	}})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

// withOtherLinks differs from withLinks, so the pair describes a real
// replacement rather than an unrelated edit.
func withOtherLinks(t *testing.T) types.List {
	t.Helper()
	list, diags := FrameworksToList(context.Background(), []Framework{{
		Name:     types.StringValue("framework"),
		Section:  types.StringValue("other section"),
		Priority: types.StringValue("high"),
	}})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return list
}

func withoutLinks() types.List { return types.ListNull(FrameworkObjectType()) }

func TestReplaceFrameworks_NotReplacing_WritesOnce(t *testing.T) {
	var writes []request
	clearedButFailed, err := ReplaceFrameworks(withoutLinks(), withLinks(t),
		request{Name: "alert", Frameworks: []string{"f1"}}, clearLinks,
		func(r request) error { writes = append(writes, r); return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 1 || len(writes[0].Frameworks) != 1 {
		t.Fatalf("expected a single write carrying the links, got %+v", writes)
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false")
	}
}

// The API merges posted frameworks, so replacing one set with another needs a
// clearing write first.
func TestReplaceFrameworks_Replacing_ClearsThenWrites(t *testing.T) {
	var writes []request
	clearedButFailed, err := ReplaceFrameworks(withLinks(t), withOtherLinks(t),
		request{Name: "alert", Frameworks: []string{"f1"}}, clearLinks,
		func(r request) error { writes = append(writes, r); return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 2 {
		t.Fatalf("expected a clear and a write, got %+v", writes)
	}
	if writes[0].Frameworks != nil {
		t.Errorf("the first write must carry no links, got %+v", writes[0])
	}
	if len(writes[1].Frameworks) != 1 {
		t.Errorf("the second write must carry the new links, got %+v", writes[1])
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false on full success")
	}
}

func TestReplaceFrameworks_Replacing_ClearFails(t *testing.T) {
	wantErr := errors.New("boom")
	var writes int
	clearedButFailed, err := ReplaceFrameworks(withLinks(t), withOtherLinks(t),
		request{Name: "alert"}, clearLinks,
		func(request) error { writes++; return wantErr },
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	if writes != 1 {
		t.Fatalf("a failed clear must not be followed by the real write, got %d writes", writes)
	}
	if clearedButFailed {
		t.Fatal("clearedButFailed should be false when the clear itself failed")
	}
}

func TestReplaceFrameworks_Replacing_WriteFailsAfterClear(t *testing.T) {
	wantErr := errors.New("boom")
	var writes int
	clearedButFailed, err := ReplaceFrameworks(withLinks(t), withOtherLinks(t),
		request{Name: "alert"}, clearLinks,
		func(r request) error {
			writes++
			if writes == 1 {
				return nil
			}
			return wantErr
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	if !clearedButFailed {
		t.Fatal("clearedButFailed should be true: the clear landed remotely, the write did not")
	}
	cleared, persist := FrameworksAfterFailedReplace(clearedButFailed)
	if !persist || cleared.IsNull() || len(cleared.Elements()) != 0 {
		t.Fatalf("state must hold an empty list after a half-applied replace: %v", cleared)
	}
}

func TestReplaceFrameworks_NoLinksEitherSide_WritesOnce(t *testing.T) {
	var writes int
	if _, err := ReplaceFrameworks(withoutLinks(), withoutLinks(),
		request{Name: "alert"}, clearLinks,
		func(request) error { writes++; return nil },
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writes != 1 {
		t.Fatalf("expected a single write, got %d", writes)
	}
	if _, persist := FrameworksAfterFailedReplace(false); persist {
		t.Fatal("nothing to persist when no clear happened")
	}
}

// An unrelated edit — a new description, say — must not clear links it is not
// changing: the clearing write would delete them until the second request lands,
// and lose them if it failed.
func TestReplaceFrameworks_UnchangedLinks_WritesOnce(t *testing.T) {
	links := withLinks(t)
	var writes []request
	clearedButFailed, err := ReplaceFrameworks(links, links,
		request{Name: "renamed", Frameworks: []string{"f1"}}, clearLinks,
		func(r request) error { writes = append(writes, r); return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writes) != 1 {
		t.Fatalf("unchanged links must not be cleared first, got %d writes: %+v", len(writes), writes)
	}
	if len(writes[0].Frameworks) != 1 {
		t.Errorf("the single write must carry the links, got %+v", writes[0])
	}
	if clearedButFailed {
		t.Error("nothing was cleared")
	}
}
