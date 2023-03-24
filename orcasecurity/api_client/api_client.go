package api_client

import (
	"fmt"
	"io/ioutil"
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

func (c *APIClient) doRequest(req http.Request) ([]byte, error) {
	req.Header.Set("authorization", fmt.Sprintf("Token %s", c.APIToken))
	res, err := c.HTTPClient.Do(&req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d", res.StatusCode)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, err
}
