package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type JiraTemplate struct {
	ID           string `json:"id"`
	TemplateName string `json:"template_name"`
}

func (client *APIClient) GetJiraTemplate(ID string) (*JiraTemplate, error) {
	type responseType struct {
		Data []JiraTemplate `json:"data"`
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/external_service/config", client.APIEndpoint), nil)
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
	return &response.Data[0], nil
}
