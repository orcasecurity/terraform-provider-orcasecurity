package api_client

import (
	"fmt"
)

const crownJewelsAPIPath = "/api/attack_paths/crown_jewels"

// CrownJewel is a user-defined (user-marked) crown jewel.
type CrownJewel struct {
	GroupUniqueID  string `json:"group_unique_id"`
	Description    string `json:"description,omitempty"`
	Severity       int    `json:"severity,omitempty"`
	CreateTime     string `json:"create_time,omitempty"`
	UpdateTime     string `json:"update_time,omitempty"`
	UserEmail      string `json:"user_email,omitempty"`
	LastUserAction string `json:"last_user_action,omitempty"`
}

type crownJewelWriteRequest struct {
	GroupUniqueIDs []string `json:"group_unique_ids"`
	Description    string   `json:"description,omitempty"`
}

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

func (client *APIClient) CreateCrownJewel(data CrownJewel) (*CrownJewel, error) {
	if err := client.upsertCrownJewel(data); err != nil {
		return nil, err
	}
	return client.getCrownJewelOrError(data.GroupUniqueID, "create")
}

func (client *APIClient) UpdateCrownJewel(data CrownJewel) (*CrownJewel, error) {
	if err := client.upsertCrownJewel(data); err != nil {
		return nil, err
	}
	return client.getCrownJewelOrError(data.GroupUniqueID, "update")
}

func (client *APIClient) DeleteCrownJewel(groupUniqueID string) error {
	_, err := client.DeleteWithBody(crownJewelsAPIPath, crownJewelWriteRequest{
		GroupUniqueIDs: []string{groupUniqueID},
	})
	return err
}

func (client *APIClient) upsertCrownJewel(data CrownJewel) error {
	_, err := client.Post(crownJewelsAPIPath, crownJewelWriteRequest{
		GroupUniqueIDs: []string{data.GroupUniqueID},
		Description:    data.Description,
	})
	return err
}

func (client *APIClient) getCrownJewelOrError(groupUniqueID, op string) (*CrownJewel, error) {
	jewel, err := client.GetCrownJewel(groupUniqueID)
	if err != nil {
		return nil, err
	}
	if jewel == nil {
		return nil, fmt.Errorf("crown jewel %q not found after %s", groupUniqueID, op)
	}
	return jewel, nil
}
