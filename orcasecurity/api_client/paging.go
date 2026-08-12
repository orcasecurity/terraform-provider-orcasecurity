package api_client

import (
	"fmt"
	"net/url"
	"strconv"
)

// data is *[]T — missing key is an error. total_items is intentionally not modeled:
// pagination terminates on an empty page (see paginateOffset), not on a reported total.
type offsetEnvelope[T any] struct {
	Data *[]T `json:"data"`
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

// Page until an empty page; maxPages guards a server that never returns one.
func paginateOffset[T any](client *APIClient, path string, filters listFilters, limit, maxPages int) ([]T, error) {
	var all []T
	for page := 0; ; page++ {
		if page >= maxPages {
			return nil, fmt.Errorf("%s: exceeded %d pages; aborting to avoid unbounded paging", path, maxPages)
		}
		query := filters.query()
		query.Set("limit", strconv.Itoa(limit))
		// Offset by rows accumulated, not page*limit — DRF clamps limit, so advancing by
		// the requested limit would skip rows the server declined to serve.
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
		// Short page ≠ done (limit may be clamped) — only an empty page means done.
		// DRF's LimitOffsetPagination always yields an empty page past the real
		// set regardless of what count reports, so total_items isn't needed for
		// loop control and isn't trusted for it either.
		if len(data) == 0 {
			return all, nil
		}
	}
}
