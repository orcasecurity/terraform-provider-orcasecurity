package shift_left_integration

import (
	"errors"
	"testing"
)

func TestDeleteByLookup_EmptyIDNotFound(t *testing.T) {
	err := DeleteByLookup(
		"",
		func() (*struct{ ID string }, error) { return nil, nil },
		func(u *struct{ ID string }) string { return u.ID },
		func(string) error { t.Fatal("DELETE must not run"); return nil },
	)
	if !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("expected ErrUnitNotFound, got %v", err)
	}
}

func TestDeleteByLookup_ResolvesIDAndDeletes(t *testing.T) {
	var deleted string
	err := DeleteByLookup(
		"",
		func() (*struct{ ID string }, error) { return &struct{ ID string }{ID: "unit-1"}, nil },
		func(u *struct{ ID string }) string { return u.ID },
		func(id string) error { deleted = id; return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != "unit-1" {
		t.Fatalf("expected delete of unit-1, got %q", deleted)
	}
}

func TestDeleteByLookup_UsesProvidedID(t *testing.T) {
	lookups := 0
	var deleted string
	err := DeleteByLookup(
		"unit-9",
		func() (*struct{ ID string }, error) { lookups++; return nil, nil },
		func(u *struct{ ID string }) string { return u.ID },
		func(id string) error { deleted = id; return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if lookups != 0 {
		t.Fatal("lookup must be skipped when id is already known")
	}
	if deleted != "unit-9" {
		t.Fatalf("expected delete of unit-9, got %q", deleted)
	}
}
