package api_client

import (
	"encoding/json"
	"fmt"
)

// scmEnvelope is the common enveloped list response for shift-left SCM endpoints.
type scmEnvelope[T any] struct {
	TotalItems int `json:"total_items"`
	Data       []T `json:"data"`
}

// getAllScmPages fetches every page of an enveloped {total_items,data} list.
// basePath must already include a leading "/api" and no query string.
// Uses limit/offset; offset advances by items actually received.
func getAllScmPages[T any](client *APIClient, basePath string) ([]T, error) {
	const pageLimit = 200
	var all []T
	for {
		sep := "?"
		if containsRune(basePath, '?') {
			sep = "&"
		}
		resp, err := client.Get(fmt.Sprintf("%s%slimit=%d&offset=%d", basePath, sep, pageLimit, len(all)))
		if err != nil {
			return nil, err
		}
		var env scmEnvelope[T]
		if err := json.Unmarshal(resp.Body(), &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if len(env.Data) == 0 || len(all) >= env.TotalItems {
			return all, nil
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
