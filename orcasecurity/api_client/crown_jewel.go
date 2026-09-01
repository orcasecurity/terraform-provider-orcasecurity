package api_client

import (
	"fmt"
	"strings"
	"time"
)

const crownJewelsAPIPath = "/api/attack_paths/crown_jewels"

const detectedCrownJewelThreshold = 20

const servingLayerQueryPath = "/api/serving-layer/query"

// DefaultCrownJewelTimeout is the create/update/delete HTTP timeout when the
// resource timeouts block is omitted. POST/DELETE return only after attack-path
// score and inventory sync, which often exceeds defaultHTTPTimeout. Combined
// with withoutTimeoutRetry so a slow write is not replayed up to 5×.
const DefaultCrownJewelTimeout = 60 * time.Second

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

func (client *APIClient) crownJewelWriteClient(timeout time.Duration) *APIClient {
	if timeout <= 0 {
		timeout = DefaultCrownJewelTimeout
	}
	return client.withHTTPTimeout(timeout).withoutTimeoutRetry()
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

// InventoryGroup is a serving-layer inventory row keyed by group_unique_id.
type InventoryGroup struct {
	GroupUniqueID           string
	DetectedCrownJewelScore int
	IsCrownJewel            bool
}

// IsOrcaDetected is true when the analyzer score meets the engine threshold
// (same cutoff as RestCrownJewels / DETECTED_CROWN_JEWEL_THRESHOLD).
func (g *InventoryGroup) IsOrcaDetected() bool {
	return g != nil && g.DetectedCrownJewelScore >= detectedCrownJewelThreshold
}

type servingLayerQueryRequest struct {
	Query  servingLayerObjectSet `json:"query"`
	Limit  int                   `json:"limit"`
	Select []string              `json:"select"`
}

type servingLayerObjectSet struct {
	Models []string           `json:"models"`
	Type   string             `json:"type"`
	With   servingLayerFilter `json:"with"`
}

type servingLayerFilter struct {
	Key      string   `json:"key"`
	Type     string   `json:"type"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type servingLayerQueryResponse struct {
	Data []servingLayerRow `json:"data"`
}

type servingLayerRow struct {
	GroupUniqueID string            `json:"group_unique_id"`
	Data          servingLayerAttrs `json:"data"`
}

type servingLayerAttrs struct {
	DetectedCrownJewelScore servingLayerInt  `json:"DetectedCrownJewelScore"`
	IsCrownJewel            servingLayerBool `json:"IsCrownJewel"`
}

type servingLayerInt struct {
	Value *int `json:"value"`
}

type servingLayerBool struct {
	Value *bool `json:"value"`
}

// GetInventoryGroup looks up one inventory group via serving-layer query.
// Returns nil, nil when the id is not in inventory.
func (client *APIClient) GetInventoryGroup(groupUniqueID string) (*InventoryGroup, error) {
	resp, err := client.Post(servingLayerQueryPath, servingLayerQueryRequest{
		Query: servingLayerObjectSet{
			Models: []string{"Inventory"},
			Type:   "object_set",
			With: servingLayerFilter{
				Key:      "GroupUniqueId",
				Type:     "str",
				Operator: "eq",
				Values:   []string{groupUniqueID},
			},
		},
		Limit:  1,
		Select: []string{"GroupUniqueId", "DetectedCrownJewelScore", "IsCrownJewel"},
	})
	if err != nil {
		return nil, err
	}
	var payload servingLayerQueryResponse
	if err := resp.ReadJSON(&payload); err != nil {
		return nil, err
	}
	if len(payload.Data) == 0 {
		return nil, nil
	}
	row := payload.Data[0]
	g := &InventoryGroup{GroupUniqueID: row.GroupUniqueID}
	if g.GroupUniqueID == "" {
		g.GroupUniqueID = groupUniqueID
	}
	if row.Data.DetectedCrownJewelScore.Value != nil {
		g.DetectedCrownJewelScore = *row.Data.DetectedCrownJewelScore.Value
	}
	if row.Data.IsCrownJewel.Value != nil {
		g.IsCrownJewel = *row.Data.IsCrownJewel.Value
	}
	return g, nil
}

// InventoryGroupExists reports whether inventory contains this group_unique_id.
func (client *APIClient) InventoryGroupExists(groupUniqueID string) (bool, error) {
	g, err := client.GetInventoryGroup(groupUniqueID)
	if err != nil {
		return false, err
	}
	return g != nil, nil
}

// SetCrownJewel marks an asset as a user-defined crown jewel (create or update).
// POST is a synchronous upsert; the follow-up GET is a read-your-writes check.
// description must be non-empty (UI "Reason"): omitting it stores null and breaks
// subsequent list/read of crown jewels on the API.
// timeout is the per-request HTTP timeout (resource timeouts block); <=0 uses DefaultCrownJewelTimeout.
func (client *APIClient) SetCrownJewel(groupUniqueID, description string, timeout time.Duration) (*CrownJewel, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("description (Reason) must be non-empty")
	}

	c := client.crownJewelWriteClient(timeout)
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
func (client *APIClient) DeleteCrownJewel(groupUniqueID string, timeout time.Duration) error {
	_, err := client.crownJewelWriteClient(timeout).DeleteWithBody(crownJewelsAPIPath, crownJewelDeleteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
	})
	return err
}
