package api_client

import (
	"fmt"
)

const crownJewelsAPIPath = "/api/attack_paths/crown_jewels"

// CrownJewel is a user-defined (user-marked) crown jewel.
type CrownJewel struct {
	GroupUniqueID string `json:"group_unique_id"`
	Description   string `json:"description"`
}

type crownJewelWriteRequest struct {
	GroupUniqueIDs []string `json:"group_unique_ids"`
	Description    string   `json:"description,omitempty"`
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
// Always pass a non-empty description (UI "Reason"): omitting it stores null and
// breaks subsequent list/read of crown jewels on the API.
func (client *APIClient) SetCrownJewel(groupUniqueID, description string) (*CrownJewel, error) {
	if _, err := client.Post(crownJewelsAPIPath, crownJewelWriteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
		Description:    description,
	}); err != nil {
		return nil, err
	}

	jewel, err := client.GetCrownJewel(groupUniqueID)
	if err != nil {
		return nil, err
	}
	if jewel == nil {
		return nil, fmt.Errorf("crown jewel %q not found after write", groupUniqueID)
	}
	return jewel, nil
}

// DeleteCrownJewel unsets a user-defined crown jewel. A missing id surfaces as
// an API error (same majority pattern as other resources); we do not treat 404
// as already-gone. Closest structural analogue (rbac deleteAccess) does swallow
// 404 — deliberately not matched here.
func (client *APIClient) DeleteCrownJewel(groupUniqueID string) error {
	_, err := client.DeleteWithBody(crownJewelsAPIPath, crownJewelWriteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
	})
	return err
}
