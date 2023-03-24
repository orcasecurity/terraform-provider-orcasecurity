package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type RBACGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SSOGroup    bool   `json:"sso_group"`
}

func (client *APIClient) GetRBACGroups() ([]RBACGroup, error) {
	type responseType struct {
		Status string `json:"status"`
		Data   struct {
			Groups []RBACGroup `json:"groups"`
		} `json:"data"`
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/rbac/group", client.APIEndpoint), nil)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.Data.Groups, err
}

func (client *APIClient) GetRBACGroup(groupID string) (*RBACGroup, error) {
	type responseType struct {
		Data RBACGroup `json:"data"`
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/rbac/group/%s", client.APIEndpoint, groupID), nil)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) CreateRBACGroup(group RBACGroup) (*RBACGroup, error) {
	type responseType struct {
		Data RBACGroup `json:"data"`
	}

	payload, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/rbac/group", client.APIEndpoint),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data, nil
}
