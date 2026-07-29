package api_client

import (
	"fmt"
	"net/url"
	"strconv"
)

// total_items is *int — absent must not read as 0. data is *[]T — missing key is an error.
type offsetEnvelope[T any] struct {
	TotalItems *int `json:"total_items"`
	Data       *[]T `json:"data"`
}

// Unknown filter keys are silently ignored — always verify matches locally.
type listFilters map[string]string

func (f listFilters) query() url.Values {
	q := url.Values{}
	for key, value := range f {
		q.Set(key, value)
	}
	return q
}

// paginateOffset walks a limit/start_at_index-paginated endpoint until an empty
// page or a known total is reached. maxPages guards against unbounded paging
// from an inflated or bogus total_items.
func paginateOffset[T any](client *APIClient, path string, filters listFilters, limit, maxPages int) ([]T, error) {
	var all []T
	for page := 0; ; page++ {
		if page >= maxPages {
			return nil, fmt.Errorf("%s: exceeded %d pages; aborting to avoid unbounded paging", path, maxPages)
		}
		query := filters.query()
		query.Set("limit", strconv.Itoa(limit))
		query.Set("start_at_index", strconv.Itoa(len(all)))
		resp, err := client.Get(path + "?" + query.Encode())
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
