package api_client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGetTrustedCloudAccount_TransportErrorDoesNotPanic(t *testing.T) {
	stubRetrySleep(t)
	httpClient := &http.Client{Transport: errRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset by peer")
	})}
	c := &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	got, err := c.GetTrustedCloudAccount("7")
	if err == nil || got != nil {
		t.Fatalf("got %+v, %v; want nil and the transport error", got, err)
	}
}

func TestCreateScmInstallation_UndecodableBodyIsAmbiguous(t *testing.T) {
	httpClient := &http.Client{Transport: errRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`<html>ok</html>`)), Header: make(http.Header), Request: req}, nil
	})}
	c := &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	_, err := createScmInstallation[struct{ ID string }](c, "/api/x/", map[string]string{})
	if !errors.Is(err, ErrAmbiguousWrite) {
		t.Fatalf("err = %v, want ErrAmbiguousWrite", err)
	}
}

func TestExternalServiceConfigCreate_UndecodableBodyIsAmbiguous(t *testing.T) {
	httpClient := &http.Client{Transport: errRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201, Body: io.NopCloser(strings.NewReader(`<html>ok</html>`)), Header: make(http.Header), Request: req}, nil
	})}
	c := &APIClient{APIEndpoint: retryTestAPIEndpoint, APIToken: "secret", HTTPClient: httpClient}
	_, err := ExternalServiceConfigCreateOp(c, OpsgenieServiceName, OpsgenieExternalServiceConfig{TemplateName: "tf"}).Create()
	if !errors.Is(err, ErrAmbiguousWrite) {
		t.Fatalf("err = %v, want ErrAmbiguousWrite", err)
	}
}
