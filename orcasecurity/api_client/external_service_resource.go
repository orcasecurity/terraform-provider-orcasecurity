package api_client

const externalServiceResourcesPath = "/api/external_service/resources"

// externalServiceResourceCreateOp wraps a /api/external_service/resources create. Names are not
// unique there, so a replayed create silently adds a second row.
func externalServiceResourceCreateOp[T any](what string, create func() (*T, error), list func() ([]T, error), id func(*T) string, match func(*T) bool, get func(string) (*T, error)) CreateOp[T] {
	return CreateOp[T]{
		What:   what,
		Create: create,
		IDs: func() (map[string]bool, error) {
			all, err := list()
			if err != nil {
				return nil, err
			}
			return idsWhere(all, id, match), nil
		},
		Get: get,
	}
}
