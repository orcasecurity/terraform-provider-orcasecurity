package api_client

import (
	"net/http"
	"time"
)

type APIClient struct {
	APIEndpoint string
	APIToken    string
	HTTPClient  *http.Client
}

func NewAPIClient(endpoint, token *string) (*APIClient, error) {
	apiclient := APIClient{
		APIEndpoint: *endpoint,
		APIToken:    *token,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
	return &apiclient, nil
}
