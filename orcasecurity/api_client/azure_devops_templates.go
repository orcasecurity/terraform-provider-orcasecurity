package api_client

import (
	"fmt"
	"net/url"
)

const AzureDevopsServiceConfigName = "azure_devops"

type AzureDevopsTemplate struct {
	ID           string `json:"id"`
	TemplateName string `json:"template_name"`
}

func (client *APIClient) GetAzureDevopsTemplateByName(name string) (*AzureDevopsTemplate, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/external_service/config?service_name=%s&template_name=%s",
			AzureDevopsServiceConfigName, url.QueryEscape(name),
		),
	)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data []AzureDevopsTemplate `json:"data"`
	}

	response := responseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("azure Devops template named '%s' does not exist", name)
	}

	if len(response.Data) > 1 {
		return nil, fmt.Errorf("there are too many results for template '%s'. expected one", name)
	}

	return &response.Data[0], nil
}
