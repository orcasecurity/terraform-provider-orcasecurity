package api_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	APIEndpoint string
	APIToken    string
	HTTPClient  *http.Client
}

// Returns an API client.
func NewAPIClient(endpoint, token *string) (*APIClient, error) {
	apiclient := APIClient{
		APIEndpoint: *endpoint,
		APIToken:    *token,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
	return &apiclient, nil
}

// A wrapper over http.Response.
type APIResponse struct {
	_body    []byte
	response *http.Response
}

// Returns response status code
func (resp *APIResponse) StatusCode() int {
	return resp.response.StatusCode
}

// Tests if the request had a status code <400 (implying it was successful).
func (resp *APIResponse) IsOk() bool {
	return resp.StatusCode() < 400
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
	// Log request details
	fmt.Printf("Making request to: %s %s\n", req.Method, req.URL)

	res, err := c.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Log response details
	fmt.Printf("Response Status: %d\n", res.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))

	response := APIResponse{
		_body:    body,
		response: res,
	}

	if !response.IsOk() {
		err := response.Error()
		return &response, fmt.Errorf("API returned error - status: %d, body: %s, error: %v",
			res.StatusCode, string(body), err)
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

// Execute HEAD HTTP request
func (c *APIClient) Head(path string) (*APIResponse, error) {
	req, err := http.NewRequest("HEAD", fmt.Sprintf("%s%s", c.APIEndpoint, path), nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(*req)
}

// Execute POST HTTP request.
func (c *APIClient) Post(path string, data interface{}) (*APIResponse, error) {
	payload, err := json.Marshal(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	fullURL := fmt.Sprintf("%s%s", c.APIEndpoint, path)
	fmt.Printf("Making POST request to: %s\n", fullURL) // Debug log

	req, err := http.NewRequest(
		"POST",
		fullURL,
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v, URL: %s", err, fullURL)
	}

	// Add debug logging for request payload
	fmt.Printf("Request payload: %s\n", string(payload))

	response, err := c.doRequest(*req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v, URL: %s, payload: %s", err, fullURL, string(payload))
	}

	return response, nil
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
