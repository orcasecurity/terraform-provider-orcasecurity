package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type AutomationFilter struct {
	Field    string   `json:"field"`
	Includes []string `json:"includes"`
}

type AutomationQuery struct {
	Filter []AutomationFilter `json:"filter"`
}

type Automation struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Query       AutomationQuery `json:"dsl_filter"`
}

type automationAPIResponseType struct {
	Data Automation `json:"data"`
}

func (client *APIClient) GetAutomation(automationID string) (*Automation, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/rules/%s", client.APIEndpoint, automationID), nil)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) CreateAutomation(automation Automation) (*Automation, error) {
	payload, err := json.Marshal(automation)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/api/rules", client.APIEndpoint),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil

}

func (client *APIClient) UpdateAutomation(ID string, data Automation) (*Automation, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/api/rules/%s", client.APIEndpoint, ID),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	body, err := client.doRequest(*req)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAutomation(ID string) error {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/rules/%s", client.APIEndpoint, ID),
		nil,
	)
	if err != nil {
		return err
	}

	_, err = client.doRequest(*req)
	if err != nil {
		return err
	}
	return nil
}
