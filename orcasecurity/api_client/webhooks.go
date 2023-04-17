package api_client

import (
	"fmt"
	"net/url"
)

const WebhookServiceConfigName = "webhook"

type Webhook struct {
	ID           string `json:"id"`
	TemplateName string `json:"template_name"`
}

func (client *APIClient) GetWebhookByName(name string) (*JiraTemplate, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/external_service/config?service_name=%s&template_name=%s",
			WebhookServiceConfigName, url.QueryEscape(name),
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
		return nil, fmt.Errorf("webhook named '%s' does not exists", name)
	}

	if len(response.Data) > 1 {
		return nil, fmt.Errorf("too many results for webhook '%s'. expected one", name)
	}

	return &response.Data[0], nil
}
