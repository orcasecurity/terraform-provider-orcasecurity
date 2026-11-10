package api_client

import "strings"

// CreateOp is a create sent with PostCreate (never replayed after an ambiguous outcome) plus what
// createrecovery.Run needs to find the object if that POST may have committed anyway.
type CreateOp[T any] struct {
	What   string
	Create func() (*T, error)
	// IDs returns the ids of every existing object matching the create's natural key. It is taken
	// before and after the POST; only an id absent from the first set is ever adopted.
	IDs func() (map[string]bool, error)
	Get func(id string) (*T, error)
}

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
