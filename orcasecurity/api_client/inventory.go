package api_client

const servingLayerQueryPath = "/api/serving-layer/query"

// InventoryGroup is a serving-layer inventory row keyed by group_unique_id.
type InventoryGroup struct {
	GroupUniqueID string
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
	GroupUniqueID string `json:"group_unique_id"`
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
		Select: []string{"GroupUniqueId"},
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
	g := &InventoryGroup{GroupUniqueID: payload.Data[0].GroupUniqueID}
	if g.GroupUniqueID == "" {
		g.GroupUniqueID = groupUniqueID
	}
	return g, nil
}
