package api_client

import (
	"fmt"
	"strings"
	"time"
)

const crownJewelsAPIPath = "/api/attack_paths/crown_jewels"

// crownJewelHTTPTimeout covers POST/DELETE on real assets: the API returns only
// after attack-path score and inventory sync, which often exceeds the shared
// defaultHTTPTimeout. Scoped here so other resources keep the shorter default.
// Combined with withoutTimeoutRetry so a slow write is not replayed up to 5×.
const crownJewelHTTPTimeout = 60 * time.Second

// CrownJewel is a user-defined (user-marked) crown jewel.
type CrownJewel struct {
	GroupUniqueID string `json:"group_unique_id"`
	Description   string `json:"description"`
}

type crownJewelWriteRequest struct {
	GroupUniqueIDs []string `json:"group_unique_ids"`
	Description    string   `json:"description"`
}

// crownJewelDeleteRequest omits description: DELETE must not send an empty
// description field (the API treats blank/null description as harmful on write paths).
type crownJewelDeleteRequest struct {
	GroupUniqueIDs []string `json:"group_unique_ids"`
}

func (client *APIClient) crownJewelWriteClient() *APIClient {
	return client.withHTTPTimeout(crownJewelHTTPTimeout).withoutTimeoutRetry()
}

// GetCrownJewel looks up one user-defined crown jewel by group_unique_id.
//
// The crown-jewels list endpoint is a single unpaginated GET that returns every
// row and ignores query params — do not send limit/start_at_index. Live probe
// and RestCrownJewels.get both return the full list in one response.
func (client *APIClient) GetCrownJewel(groupUniqueID string) (*CrownJewel, error) {
	resp, err := client.Get(crownJewelsAPIPath)
	if err != nil {
		return nil, err
	}

	var jewels []CrownJewel
	if err = resp.ReadJSON(&jewels); err != nil {
		return nil, err
	}
	for i := range jewels {
		if jewels[i].GroupUniqueID == groupUniqueID {
			return &jewels[i], nil
		}
	}
	return nil, nil
}

// SetCrownJewel marks an asset as a user-defined crown jewel (create or update).
// POST is a synchronous upsert; the follow-up GET is a read-your-writes check.
// description must be non-empty (UI "Reason"): omitting it stores null and breaks
// subsequent list/read of crown jewels on the API.
//
// The API does not validate that group_unique_id exists in inventory — a typo
// still creates a CrownJewel row that the refetch GET will find.
func (client *APIClient) SetCrownJewel(groupUniqueID, description string) (*CrownJewel, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("description (Reason) must be non-empty")
	}

	c := client.crownJewelWriteClient()
	if _, err := c.Post(crownJewelsAPIPath, crownJewelWriteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
		Description:    description,
	}); err != nil {
		return nil, err
	}

	jewel, err := c.GetCrownJewel(groupUniqueID)
	if err != nil {
		return nil, err
	}
	if jewel == nil {
		return nil, fmt.Errorf("crown jewel %q not found after write", groupUniqueID)
	}
	return jewel, nil
}

// DeleteCrownJewel disables a user-defined crown jewel the same way the Orca UI
// does (DELETE /attack_paths/crown_jewels). That is not a hard delete: the API
// upserts is_crown_jewel=false (an active "not a crown jewel" override). Calling
// DELETE on an unmarked id returns 200 and may create such a row — the endpoint
// does not 404.
func (client *APIClient) DeleteCrownJewel(groupUniqueID string) error {
	_, err := client.crownJewelWriteClient().DeleteWithBody(crownJewelsAPIPath, crownJewelDeleteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
	})
	return err
}
