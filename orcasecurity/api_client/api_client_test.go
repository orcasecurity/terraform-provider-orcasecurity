package api_client

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestAPIResponse_StatusCode(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")
	if resp.StatusCode() != 200 {
		t.Errorf("expected status code 200, got %d", resp.StatusCode())
	}
}
func TestAPIResponse_IsOk_Success(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")
	if !resp.IsOk() {
		t.Error("expected IsOk to return true, got false")
	}
}
func TestAPIResponse_IsOk_Failure(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")
	if resp.IsOk() {
		t.Error("expected IsOk to return false, got true")
	}
}

func TestAPIResponse_Body(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	body := resp.Body()
	if string(body) != "ok" {
		t.Errorf("expected body to be 'ok', got %s", body)
	}
}

func TestAPIResponse_Read(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	body, err := resp.Read()
	if err != nil {
		t.Errorf("expected response to read body, got error: %s", err)
	}
	if string(body) != "ok" {
		t.Errorf("expected to read 'ok', got %s", body)
	}
}

func TestAPIResponse_ReadJSON(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{"status": "ok"}`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	data := struct {
		Status string `json:"status"`
	}{}

	err := resp.ReadJSON(&data)
	if err != nil {
		t.Errorf("expected response to read JSON body, got error: %s", err)
	}
	if data.Status != "ok" {
		t.Errorf("expected to read 'ok', got %s", data.Status)
	}
}

func TestAPIResponse_Error_FromMessage(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader(`{"message": "not ok"}`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	err := resp.Error()
	if err.Error() != "not ok" {
		t.Errorf("expected to read 'ok', got %s", err.Error())
	}
}

func TestAPIResponse_Error_FromError(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error": "not ok"}`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	err := resp.Error()
	if err.Error() != "not ok" {
		t.Errorf("expected to read 'ok', got %s", err.Error())
	}
}

func TestAPIResponse_Error_ReturnNil(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	err := resp.Error()
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err.Error())
	}
}

func TestAPIClient_AddAuthorizationHeader(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	if resp.response.Request.Header.Get("authorization") != "Token secret" {
		t.Error("expected authorization header")
	}
}

func TestAPIClient_AddContentTypeHeader(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	if resp.response.Request.Header.Get("content-type") != "application/json" {
		t.Error("expected content-type header")
	}
}

func TestAPIClient_AddUserAgentHeader(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/")

	if resp.response.Request.Header.Get("user-agent") != "orca-terraform-provider (+https://registry.terraform.io/providers/orcasecurity)" {
		t.Error("expected content-type header")
	}
}

func TestAPIClient_Get(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Get("/users")

	body := resp.Body()
	if string(body) != "ok" {
		t.Errorf("expected to read 'ok', got %s", body)
	}

	if resp.response.Request.Method != "GET" {
		t.Errorf("expected GET, got %s", resp.response.Request.Method)
	}

	if resp.response.Request.URL.String() != "http://localhost/users" {
		t.Errorf("expected http://localhost/users, got %+v", resp.response.Request.URL.String())
	}
}
func TestAPIClient_Post(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 201,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	data := struct {
		ID string `json:"id"`
	}{ID: "42"}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Post("/users", &data)

	body := resp.Body()
	if string(body) != "ok" {
		t.Errorf("expected to read 'ok', got %s", body)
	}

	if resp.response.Request.Method != "POST" {
		t.Errorf("expected POST, got %s", resp.response.Request.Method)
	}

	if resp.response.Request.URL.String() != "http://localhost/users" {
		t.Errorf("expected http://localhost/users, got %+v", resp.response.Request.URL.String())
	}

	sent_body, err := io.ReadAll(resp.response.Request.Body)
	// sent_body, err := resp.response.Request.Body.Read()
	if err != nil {
		t.Error(err)
	}

	if string(sent_body) != `{"id":"42"}` {
		t.Errorf(`expected POST body to be b'{"id":"42"}', got %s`, sent_body)
	}
}

func TestAPIClient_Put(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	data := struct {
		ID string `json:"id"`
	}{ID: "42"}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Put("/users", &data)

	body := resp.Body()
	if string(body) != "ok" {
		t.Errorf("expected to read 'ok', got %s", body)
	}

	if resp.response.Request.Method != "PUT" {
		t.Errorf("expected PUT, got %s", resp.response.Request.Method)
	}

	if resp.response.Request.URL.String() != "http://localhost/users" {
		t.Errorf("expected http://localhost/users, got %+v", resp.response.Request.URL.String())
	}

	sent_body, err := io.ReadAll(resp.response.Request.Body)
	// sent_body, err := resp.response.Request.Body.Read()
	if err != nil {
		t.Error(err)
	}

	if string(sent_body) != `{"id":"42"}` {
		t.Errorf(`expected PUT body to be b'{"id":"42"}', got %s`, sent_body)
	}
}
func TestAPIClient_Delete(t *testing.T) {
	httpClient := &http.Client{Transport: RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 204,
			Body:       ioutil.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	resp, _ := apiClient.Delete("/users")

	if resp.response.Request.Method != "DELETE" {
		t.Errorf("expected PUT, got %s", resp.response.Request.Method)
	}

	if resp.response.Request.URL.String() != "http://localhost/users" {
		t.Errorf("expected http://localhost/users, got %+v", resp.response.Request.URL.String())
	}
}
