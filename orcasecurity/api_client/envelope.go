package api_client

import "fmt"

// readData decodes the standard {"status": ..., "data": ...} response
// envelope and returns the data value. A missing or null data key is an
// error, never a silent zero value — callers must not write a zero-value
// struct into Terraform state.
func readData[T any](resp *APIResponse) (*T, error) {
	var envelope struct {
		Data *T `json:"data"`
	}
	if err := resp.ReadJSON(&envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("could not decode response: missing data key: %s", string(resp.Body()))
	}
	return envelope.Data, nil
}
