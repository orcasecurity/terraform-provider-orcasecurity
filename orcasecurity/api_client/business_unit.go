package api_client

import (
	"encoding/json"
	"fmt"
)

const BusinessUnitTypeCombinedFilter = "combined_filter"

// BusinessUnit is the /api/business_units representation. Config holds the
// intermediate rule tree ({"or": [{"some": {"CloudProviders": ["aws"]}}]}).
type BusinessUnit struct {
	ID                  string          `json:"id,omitempty"`
	Name                string          `json:"name"`
	BUType              string          `json:"bu_type,omitempty"`
	Config              json.RawMessage `json:"config,omitempty"`
	GlobalFilter        *bool           `json:"global_filter,omitempty"`
	BusinessCriticality string          `json:"business_criticality"`
	OwnerTeam           string          `json:"owner_team"`
	Application         string          `json:"application"`
	ContactEmails       []string        `json:"contact_emails"`
	DeploymentStages    []string        `json:"deployment_stages"`
}

// GetBusinessUnit returns nil, nil when the business unit does not exist.
func (client *APIClient) GetBusinessUnit(businessUnitID string) (*BusinessUnit, error) {
	resp, err := client.Get(fmt.Sprintf("/api/business_units/%s", businessUnitID))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return readData[BusinessUnit](resp)
}

func (client *APIClient) CreateBusinessUnit(data BusinessUnit) (*BusinessUnit, error) {
	resp, err := client.Post("/api/business_units", data)
	if err != nil {
		return nil, err
	}
	return readData[BusinessUnit](resp)
}

func (client *APIClient) UpdateBusinessUnit(ID string, data BusinessUnit) (*BusinessUnit, error) {
	resp, err := client.Put(fmt.Sprintf("/api/business_units/%s", ID), data)
	if err != nil {
		return nil, err
	}
	return readData[BusinessUnit](resp)
}

func (client *APIClient) DeleteBusinessUnit(ID string) error {
	resp, err := client.Delete(fmt.Sprintf("/api/business_units/%s", ID))
	if resp != nil && resp.StatusCode() == 404 {
		return nil
	}
	return err
}
