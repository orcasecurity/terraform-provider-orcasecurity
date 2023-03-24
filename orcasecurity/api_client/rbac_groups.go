package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RBACGroup struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SSOGroup    bool   `json:"sso_group"`
	AllUsers    bool   `json:"all_users"`
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
