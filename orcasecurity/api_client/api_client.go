package api_client

import (
	"encoding/json"
	"errors"
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
	type errorType struct {
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
	}

	req.Header.Set("authorization", fmt.Sprintf("Token %s", c.APIToken))
	req.Header.Set("content-type", "application/json")
	res, err := c.HTTPClient.Do(&req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == 400 {
			errorMessage := errorType{}
			err = json.Unmarshal(body, &errorMessage)
			if err != nil {
				return nil, fmt.Errorf("status: %d, %s", res.StatusCode, body)
			}
			message := errorMessage.Error
			if errorMessage.Message != "" {
				message = errorMessage.Message
			}
			return nil, errors.New(message)
		}
		return nil, fmt.Errorf("status: %d", res.StatusCode)
	}

	return body, err
}
