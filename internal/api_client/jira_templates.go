package api_client

import (
	"fmt"
	"net/url"
)

const JiraServiceConfigName = "jira"

type JiraTemplate struct {
	ID           string `json:"id"`
	TemplateName string `json:"template_name"`
}

func (client *APIClient) GetJiraTemplateByName(name string) (*JiraTemplate, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/external_service/config?service_name=%s&template_name=%s",
			JiraServiceConfigName, url.QueryEscape(name),
		),
	)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data []JiraTemplate `json:"data"`
	}

	response := responseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("jira template named '%s' does not exists", name)
	}

	if len(response.Data) > 1 {
		return nil, fmt.Errorf("too many results for template '%s'. expected one", name)
	}

	return &response.Data[0], nil
}
