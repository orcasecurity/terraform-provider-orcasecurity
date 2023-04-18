package api_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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

// Convenience wrapper over http.Response
type APIResponse struct {
	_body    []byte
	response *http.Response
}

// Return response status code
func (resp *APIResponse) StatusCode() int {
	return resp.response.StatusCode
}

// Test if request was successful (<300)
func (resp *APIResponse) IsOk() bool {
	return resp.StatusCode() < 300
}

// Read response body.
// Careful, it contains full body in the memory.
// If you wish a memory-effective version then use Execute function
// that returns pointer to raw http.Response.
func (resp *APIResponse) Body() []byte {
	return resp._body
}

// Returns either response body or response error.
// Note, the error contains the message provided by API on unsuccessful request.
func (resp *APIResponse) Read() ([]byte, error) {
	return resp.Body(), resp.Error()
}

// Load response JSON into user struct.
func (resp *APIResponse) ReadJSON(typ interface{}) error {
	return json.Unmarshal(resp.Body(), typ)
}

// Return API error message.
// Returns nil if request was successful.
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

// Perform API call.
func (c *APIClient) Execute(req http.Request) (*http.Response, error) {
	req.Header.Set("authorization", fmt.Sprintf("Token %s", c.APIToken))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "orca-terraform-provider (+https://registry.terraform.io/providers/orcasecurity)")
	return c.HTTPClient.Do(&req)
}

func (c *APIClient) doRequest(req http.Request) (*APIResponse, error) {
	res, err := c.Execute(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := APIResponse{
		_body:    body,
		response: res,
	}
	if !response.IsOk() {
		return &response, response.Error()
	}
	return &response, nil
}

// Execute GET HTTP request.
func (c *APIClient) Get(path string) (*APIResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.APIEndpoint, path), nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(*req)
}

// Execute POST HTTP request.
func (c *APIClient) Post(path string, data interface{}) (*APIResponse, error) {
	payload, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s%s", c.APIEndpoint, path),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	return c.doRequest(*req)
}

// Execute PUT HTTP request.
func (c *APIClient) Put(path string, data interface{}) (*APIResponse, error) {
	payload, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s%s", c.APIEndpoint, path),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	return c.doRequest(*req)
}

// Execute DELETE HTTP request.
func (c *APIClient) Delete(path string) (*APIResponse, error) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s", c.APIEndpoint, path), nil)
	if err != nil {
		return nil, err
	}

	return c.doRequest(*req)
}
