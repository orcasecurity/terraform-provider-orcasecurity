package api_client

import (
	"fmt"
	"net/url"
)

const WebhookServiceConfigName = "webhook"

type Webhook struct {
	ID string `json:"id"`

	Config       Config `json:"config"`
	CreatedAt    string `json:"created_at"`
	IsEnabled    bool   `json:"is_enabled"`
	TemplateName string `json:"template_name"`
}

type Config struct {
	CustomHeaders map[string]string `json:"custom_headers"`
	Type          string            `json:"type"`
	WebhookUrl    string            `json:"webhook_url"`
}

func (client *APIClient) GetWebhookByName(name string) (*Webhook, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/external_service/config?service_name=%s&template_name=%s",
			WebhookServiceConfigName, url.QueryEscape(name),
		),
	)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data []Webhook `json:"data"`
	}

	response := responseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("webhook named '%s' does not exist", name)
	}

	if len(response.Data) > 1 {
		return nil, fmt.Errorf("too many results for webhook '%s'. expected one but got %d", name, len(response.Data))
	}

	return &response.Data[0], nil
}
