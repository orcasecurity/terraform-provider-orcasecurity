package api_client

import (
	"encoding/json"
	"fmt"
)

// TDIR stands for 'Trusted Dynamic IP Range'
type TDIRVariablesType struct {
	OrgID string `json:"orgId,omitempty"`
	Value bool   `json:"value"`
}

type SetGetTDIRType struct {
	Query     string            `json:"query"`
	Variables TDIRVariablesType `json:"variables,omitempty"`
}

type OK struct {
	OK bool `json:"ok"`
}

type OuterReceptacle struct {
	Data struct {
		SetDynamicTrustedIpsEnabled OK   `json:"setDynamicTrustedIpsEnabled,omitempty"`
		IsDynamicTrustedIpsEnabled  bool `json:"isDynamicTrustedIpsEnabled,omitempty"`
	} `json:"data"`
}

func (client *APIClient) GetTrustedDynamicIpRangeStatus(orgId string) (bool, error) {
	data := SetGetTDIRType{
		Query:     "query ($orgId: String!) {isDynamicTrustedIpsEnabled(orgId: $orgId)}",
		Variables: TDIRVariablesType{OrgID: orgId},
	}
	resp, err := client.Post("/api/gql", data)
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	response := OuterReceptacle{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return false, err
	}
	return response.Data.IsDynamicTrustedIpsEnabled, nil
}

func (client *APIClient) SetTrustedDynamicIpRange(orgId string) (bool, error) {
	data := SetGetTDIRType{
		Query:     "mutation ($orgId: String!, $value: Boolean!) {setDynamicTrustedIpsEnabled(orgId: $orgId, value: $value) {ok}}",
		Variables: TDIRVariablesType{OrgID: orgId, Value: true},
	}
	resp, err := client.Post("/api/gql", data)
	if err != nil {
		return false, err
	}

	// Log the raw response
	fmt.Printf("Raw API Response: %s\n", string(resp.Body()))

	response := OuterReceptacle{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return false, fmt.Errorf("unmarshal error: %v, raw response: %s", err, string(resp.Body()))
	}

	// Log the parsed response
	fmt.Printf("Parsed Response: %+v\n", response)
	if !response.Data.SetDynamicTrustedIpsEnabled.OK {
		return false, fmt.Errorf("unexpected response format from API. Raw response: %s", string(resp.Body()))
	}

	return response.Data.SetDynamicTrustedIpsEnabled.OK, nil
}

func (client *APIClient) UnsetTrustedDynamicIpRange(orgId string) error {
	data := SetGetTDIRType{
		Query:     "mutation ($orgId: String!, $value: Boolean!) {setDynamicTrustedIpsEnabled(orgId: $orgId, value: $value) {ok}}",
		Variables: TDIRVariablesType{OrgID: orgId, Value: false},
	}

	// Debug print the struct before marshaling
	fmt.Printf("Data struct: %+v\n", data)
	fmt.Printf("Variables struct: %+v\n", data.Variables)

	// Marshal it ourselves to see what it looks like
	debugJson, _ := json.Marshal(data)
	fmt.Printf("JSON that would be sent: %s\n", string(debugJson))

	resp, err := client.Post("/api/gql", data)
	if err != nil {
		return fmt.Errorf("API post error: %v", err)
	}

	// Log the raw response for debugging
	fmt.Printf("Raw API Response for Unset: %s\n", string(resp.Body()))

	response := OuterReceptacle{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return fmt.Errorf("unmarshal error: %v, raw response: %s", err, string(resp.Body()))
	}

	if !response.Data.SetDynamicTrustedIpsEnabled.OK {
		return fmt.Errorf("API returned unsuccessful response: %s", string(resp.Body()))
	}

	return nil
}
