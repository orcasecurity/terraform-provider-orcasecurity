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

type APIResponse struct {
	_body    []byte
	response *http.Response
}

func (resp *APIResponse) StatusCode() int {
	return resp.response.StatusCode
}

func (resp *APIResponse) IsOk() bool {
	return resp.StatusCode() == http.StatusOK
}

func (resp *APIResponse) Body() []byte {
	return resp._body
}

func (resp *APIResponse) Read() ([]byte, error) {
	return resp.Body(), resp.Error()
}

func (resp *APIResponse) ReadJSON(typ interface{}) error {
	return json.Unmarshal(resp.Body(), typ)
}

func (resp *APIResponse) Error() error {
	type errorType struct {
		Message string `json:"message,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	if !resp.IsOk() {
		errorMessage := errorType{}
		err := json.Unmarshal(resp.Body(), &errorMessage)
		if err != nil {
			return fmt.Errorf("status: %d, %s", resp.StatusCode(), resp.Body())
		}
		message := errorMessage.Error
		if errorMessage.Message != "" {
			message = errorMessage.Message
		}
		return errors.New(message)
	}
	return nil
}

func (c *APIClient) doRequest(req http.Request) (*APIResponse, error) {
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

	return &APIResponse{
		_body:    body,
		response: res,
	}, nil
}

func (c *APIClient) Get(path string) (*APIResponse, error) {
	fullUrl := fmt.Sprintf("%s%s", c.APIEndpoint, path)
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(*req)
}
