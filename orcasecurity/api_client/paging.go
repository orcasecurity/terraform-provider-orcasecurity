package api_client

import "fmt"

// offsetEnvelope is the shared {"total_items":, "data":[]} shape used by every
// limit/start_at_index-paginated list endpoint. TotalItems is a pointer so an
// absent total_items is not misread as zero (which would falsely terminate
// paging after the first full page); Data is a pointer so a missing/null data
// key is an error, not a silent empty slice.
type offsetEnvelope[T any] struct {
	TotalItems *int `json:"total_items"`
	Data       *[]T `json:"data"`
}

// paginateOffset walks a limit/start_at_index-paginated endpoint until an empty
// page or a known total is reached. maxPages guards against unbounded paging
// from an inflated or bogus total_items.
func paginateOffset[T any](client *APIClient, path string, limit, maxPages int) ([]T, error) {
	var all []T
	for page := 0; ; page++ {
		if page >= maxPages {
			return nil, fmt.Errorf("%s: exceeded %d pages; aborting to avoid unbounded paging", path, maxPages)
		}
		resp, err := client.Get(fmt.Sprintf("%s?limit=%d&start_at_index=%d", path, limit, len(all)))
		if err != nil {
			return nil, err
		}
		var env offsetEnvelope[T]
		if err := resp.ReadJSON(&env); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if env.Data == nil {
			return nil, fmt.Errorf("%s: response missing data key: %s", path, resp.Body())
		}
		data := *env.Data
		all = append(all, data...)
		if len(data) == 0 || (env.TotalItems != nil && len(all) >= *env.TotalItems) {
			return all, nil
		}
	}
}
