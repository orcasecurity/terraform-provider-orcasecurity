package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
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

func (client *APIClient) createBusinessUnit(data BusinessUnit) (*BusinessUnit, error) {
	resp, err := client.PostCreate("/api/business_units", data)
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

// Names are unique per creating user, not per org, so other users' same-named units can match;
// recovery only adopts an id absent from the pre-create snapshot. search is a substring match.
func (client *APIClient) ListBusinessUnitIDsByName(name string) (map[string]bool, error) {
	all, err := paginateOffset[BusinessUnit](client, "/api/business_units", listFilters{"search": strings.TrimSpace(name)}, 300, 500)
	if err != nil {
		return nil, err
	}
	return idsWhere(all, func(b *BusinessUnit) string { return b.ID }, func(b *BusinessUnit) bool { return sameName(b.Name, name) }), nil
}

func (client *APIClient) BusinessUnitCreateOp(data BusinessUnit) CreateOp[BusinessUnit] {
	return CreateOp[BusinessUnit]{
		What:   fmt.Sprintf("business unit named %q", data.Name),
		Create: func() (*BusinessUnit, error) { return client.createBusinessUnit(data) },
		IDs:    func() (map[string]bool, error) { return client.ListBusinessUnitIDsByName(data.Name) },
		Get:    client.GetBusinessUnit,
	}
}
