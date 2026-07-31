package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

const (
	apiUsersPath   = "/api/users"
	usersPageLimit = 300
	usersMaxRows   = 1_000_000
)

// User is one row from GET /api/users (org member, not a pending invite).
type User struct {
	ID          string `json:"user_id"`
	Email       string `json:"email"`
	FirstName   string `json:"first"`
	LastName    string `json:"last"`
	Status      string `json:"status"`
	MFARequired bool   `json:"mfa_required"`
	MFAEnabled  bool   `json:"mfa_enabled"`
}

// ListUsers returns every user in the organization, paging through the offset
// paginator (limit + start_at_index query params, {status,data,total_items}
// envelope — same shape as the paginated RBAC access endpoints).
func (client *APIClient) ListUsers() ([]User, error) {
	var out []User
	fetched := 0
	for {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(usersPageLimit))
		q.Set("start_at_index", strconv.Itoa(fetched))

		resp, err := client.Get(apiUsersPath + "?" + q.Encode())
		if err != nil {
			return nil, err
		}

		var envelope struct {
			Status     string `json:"status"`
			Data       []User `json:"data"`
			TotalItems int    `json:"total_items"`
		}
		if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
			return nil, fmt.Errorf("parse user list: %w", err)
		}
		if envelope.Status != "" && envelope.Status != "success" {
			return nil, fmt.Errorf("unexpected user list status: %q", envelope.Status)
		}

		out = append(out, envelope.Data...)
		fetched += len(envelope.Data)
		if len(envelope.Data) == 0 || fetched >= envelope.TotalItems {
			return out, nil
		}
		if fetched >= usersMaxRows {
			return nil, fmt.Errorf(
				"list users: fetched %d rows without reaching total_items=%d; aborting to avoid an unbounded loop (server may be ignoring start_at_index)",
				fetched, envelope.TotalItems)
		}
	}
}
