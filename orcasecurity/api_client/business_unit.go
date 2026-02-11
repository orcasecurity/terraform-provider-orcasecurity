package api_client

import (
	"encoding/json"
	"fmt"
)

/*type BusinessUnitFilterRule struct {
	Strang string `json:"account_number"`
}*/

type BusinessUnitFilter struct {
	CloudProviders []string `json:"cloud_provider,omitempty"`
	CustomTags     []string `json:"custom_tags,omitempty"`
	CloudTags      []string `json:"inventory_tags,omitempty"`
	AccountTags    []string `json:"accounts_tags_info_list,omitempty"`
	CloudAccounts  []string `json:"cloud_vendor_id,omitempty"`
}

type BusinessUnitShiftLeftFilter struct {
	ShiftLeftProjects []string `json:"shiftleft_project_id,omitempty"`
}

type BusinessUnit struct {
	ID              string                       `json:"filter_id,omitempty"`
	Name            string                       `json:"name"`
	Filter          *BusinessUnitFilter          `json:"filter_data,omitempty"`
	ShiftLeftFilter *BusinessUnitShiftLeftFilter `json:"shiftleft_filter_data,omitempty"`
}

type businessUnitAPIResponseType struct {
	Data BusinessUnit `json:"data"`
}

func (client *APIClient) GetBusinessUnit(businessUnitID string) (*BusinessUnit, error) {
	resp, err := client.Get(fmt.Sprintf("/api/filters/%s", businessUnitID))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := businessUnitAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DoesBusinessUnitExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/filters/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateBusinessUnit(business_units BusinessUnit) (*BusinessUnit, error) {
	resp, err := client.Post("/api/filters", business_units)
	if err != nil {
		return nil, err
	}

	response := businessUnitAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateBusinessUnit(ID string, data BusinessUnit) (*BusinessUnit, error) {
	resp, err := client.Put(fmt.Sprintf("/api/filters/%s", ID), data)
	if err != nil {
		return nil, err
	}

	response := businessUnitAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteBusinessUnit(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/filters/%s", ID))
	return err
}
