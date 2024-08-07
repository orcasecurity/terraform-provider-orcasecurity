package api_client_test

import (
	"io"
	"net/http"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func TestCustomAlert_DoesCustomSonarAlertExist(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesCustomSonarAlertExist("1")
	if err != nil {
		t.Error(err)
	}
	if !exists {
		t.Error("custom alert expected to exists, but it does not")
	}

}
func TestAutomations_DoesCustomSonarAlertExist404(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesCustomSonarAlertExist("1")
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("automation expected to be absent, but it exists")
	}

}

func TestAutomations_DoesCustomSonarAlertExist500(t *testing.T) {
	httpClient := &http.Client{Transport: api_client.RoundTripFunc(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`ok`)),
			Request:    req,
		}
	})}

	apiClient := api_client.APIClient{APIEndpoint: "http://localhost", APIToken: "secret", HTTPClient: httpClient}
	exists, err := apiClient.DoesCustomSonarAlertExist("1")
	if err != nil {
		t.Error(err)
	}
	if exists {
		t.Error("automation expected to be absent, but it exists")
	}

}
