package api_client

const servingLayerQueryPath = "/api/serving-layer/query"

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
	Data []struct{} `json:"data"`
}

// InventoryGroupExists reports whether groupUniqueID is in inventory.
// Serving-layer rows are nested; existence is len(data) > 0. Select keeps the
// payload to GroupUniqueId instead of a full inventory row.
func (client *APIClient) InventoryGroupExists(groupUniqueID string) (bool, error) {
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
		return false, err
	}
	var payload servingLayerQueryResponse
	if err := resp.ReadJSON(&payload); err != nil {
		return false, err
	}
	return len(payload.Data) > 0, nil
}
