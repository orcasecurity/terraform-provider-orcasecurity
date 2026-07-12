package api_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// httpDebugEnvVar gates verbose HTTP logging. When unset, the client emits no
// request/response detail — critical because those bodies carry integration
// secrets (API tokens, keys). Set it to any non-empty value to troubleshoot.
const httpDebugEnvVar = "ORCASECURITY_HTTP_DEBUG"

// debugf logs to stderr (never stdout — stdout is the go-plugin protocol channel
// Terraform speaks over) and only when httpDebugEnvVar is set.
func (c *APIClient) debugf(format string, a ...any) {
	if os.Getenv(httpDebugEnvVar) == "" {
		return
	}
	log.Printf(format, a...)
}

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

// Test if request was successful (<400)
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
	c.debugf("Making request to: %s %s", req.Method, req.URL)

	resp, err := c.roundTripWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %v", err)
	}

	c.debugf("Response Status: %d", resp.StatusCode())
	c.debugf("Response Body: %s", string(resp.Body()))

	if !resp.IsOk() {
		apiErr := resp.Error()
		return resp, fmt.Errorf("API returned error - status: %d, body: %s, error: %v",
			resp.StatusCode(), string(resp.Body()), apiErr)
	}

	return resp, nil
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
	c.debugf("Making POST request to: %s", fullURL)

	req, err := http.NewRequest(
		"POST",
		fullURL,
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v, URL: %s", err, fullURL)
	}

	c.debugf("Request payload: %s", string(payload))

	response, err := c.doRequest(*req)
	if err != nil {
		// Do not append the request payload — it carries integration secrets and
		// this error surfaces to the user via diagnostics. doRequest already wraps
		// the server's response body for context.
		return nil, fmt.Errorf("request failed: %v, URL: %s", err, fullURL)
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

// Execute PATCH HTTP request.
func (c *APIClient) Patch(path string, data interface{}) (*APIResponse, error) {
	payload, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"PATCH",
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

// DeleteWithBody executes a DELETE HTTP request carrying a JSON body. Some Orca
// RBAC endpoints (e.g. /api/rbac/access/user) identify the record by an id in
// the request body rather than in the URL path.
func (c *APIClient) DeleteWithBody(path string, data interface{}) (*APIResponse, error) {
	payload, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s%s", c.APIEndpoint, path),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, err
	}

	return c.doRequest(*req)
}
