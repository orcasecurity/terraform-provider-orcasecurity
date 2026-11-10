package api_client

import (
	"errors"
	"fmt"
	"strings"
)

// CreateOp is a create sent with PostCreate (never replayed after an ambiguous outcome) plus what
// createrecovery.Run needs to find the object if that POST may have committed anyway.
type CreateOp[T any] struct {
	What   string
	Create func() (*T, error)
	ID     func(*T) string
	// IDs returns the ids of every existing object matching the create's natural key. It is taken
	// before and after the POST; only an id absent from the first set is ever adopted.
	IDs IDLister
	Get func(id string) (*T, error)
}

type IDLister func() (map[string]bool, error)

func sameName(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

func idsWhere[T any](rows []T, id func(*T) string, match func(*T) bool) map[string]bool {
	ids := map[string]bool{}
	for i := range rows {
		if match(&rows[i]) {
			ids[id(&rows[i])] = true
		}
	}
	return ids
}

// ReadCreated reads back an object just created as id; a failed or empty read names the id to import.
func ReadCreated[T any](what, id string, get func(string) (*T, error)) (*T, error) {
	created, err := get(id)
	if err != nil || created == nil {
		return nil, createdButUnreadableError(what, id, err)
	}
	return created, nil
}

func createdButUnreadableError(what, id string, cause error) error {
	if cause == nil {
		cause = errors.New("not found")
	}
	return fmt.Errorf("%s was created as %s, but reading it back failed (%w); import it before re-applying", what, id, cause)
}
