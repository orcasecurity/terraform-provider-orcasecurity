package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Request types

type CustomComplianceFrameworkTest struct {
	RuleID            string `json:"rule_id"`
	RuleIDInFramework string `json:"rule_id_in_framework,omitempty"`
	ControlUniqueID   string `json:"control_unique_id,omitempty"`
	Priority          string `json:"priority,omitempty"`
	OriginFrameworkID string `json:"origin_framework_id,omitempty"`
}

type CustomComplianceFrameworkSection struct {
	Name     string                             `json:"name"`
	Tests    []CustomComplianceFrameworkTest    `json:"tests"`
	Sections []CustomComplianceFrameworkSection `json:"sections"`
}

type CustomComplianceFrameworkRequest struct {
	Name               string                             `json:"name"`
	Description        *string                            `json:"description"`
	Visibility         string                             `json:"visibility,omitempty"`
	Scope              string                             `json:"scope,omitempty"`
	ForcedCloudVendors []string                           `json:"forced_cloud_vendors,omitempty"`
	Sections           []CustomComplianceFrameworkSection `json:"sections"`
}

// Response types

type CustomComplianceFrameworkWriteResponse struct {
	ID          json.Number `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
}

type customComplianceFrameworkWriteAPIResponse struct {
	Data CustomComplianceFrameworkWriteResponse `json:"data"`
}

const customComplianceFrameworkBasePath = "/api/compliance/frameworks"

// GetCustomComplianceFramework is the custom-framework alias of GetComplianceFramework.
// Only 404 is treated as gone; every other non-2xx is an error.
func (client *APIClient) GetCustomComplianceFramework(id string) (*ComplianceFramework, error) {
	return client.GetComplianceFramework(id)
}

func (client *APIClient) CreateCustomComplianceFramework(data CustomComplianceFrameworkRequest) (*CustomComplianceFrameworkWriteResponse, error) {
	resp, err := client.PostCreate(customComplianceFrameworkBasePath, data)
	if err != nil {
		return nil, err
	}

	response := customComplianceFrameworkWriteAPIResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// Names are unique per org trimmed and case-insensitive. Frameworks forced to cloud vendors the org
// has not onboarded are absent from the select map, so their recovery falls back to check-the-UI.
func (client *APIClient) ListCustomComplianceFrameworkIDsByName(name string) (map[string]bool, error) {
	selections, err := client.GetComplianceFrameworkSelections()
	if err != nil {
		return nil, err
	}
	ids := map[string]bool{}
	for id, fw := range selections {
		if fw.Custom && strings.EqualFold(strings.TrimSpace(fw.DisplayName), strings.TrimSpace(name)) {
			ids[id] = true
		}
	}
	return ids, nil
}

func (client *APIClient) CustomComplianceFrameworkCreateOp(data CustomComplianceFrameworkRequest) CreateOp[CustomComplianceFrameworkWriteResponse] {
	return CreateOp[CustomComplianceFrameworkWriteResponse]{
		What: fmt.Sprintf("custom compliance framework named %q", data.Name),
		Create: func() (*CustomComplianceFrameworkWriteResponse, error) {
			return client.CreateCustomComplianceFramework(data)
		},
		IDs: func() (map[string]bool, error) { return client.ListCustomComplianceFrameworkIDsByName(data.Name) },
		Get: func(id string) (*CustomComplianceFrameworkWriteResponse, error) {
			fw, err := client.GetCustomComplianceFramework(id)
			if err != nil || fw == nil {
				return nil, err
			}
			return &CustomComplianceFrameworkWriteResponse{ID: json.Number(fw.ID), Name: fw.DisplayName}, nil
		},
	}
}

func (client *APIClient) UpdateCustomComplianceFramework(id string, data CustomComplianceFrameworkRequest) (*CustomComplianceFrameworkWriteResponse, error) {
	resp, err := client.Put(fmt.Sprintf(customComplianceFrameworkBasePath+"/%s", id), data)
	if err != nil {
		return nil, err
	}

	response := customComplianceFrameworkWriteAPIResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCustomComplianceFramework(id string) error {
	resp, err := client.Delete(fmt.Sprintf(customComplianceFrameworkBasePath+"/%s", id))
	if isNotFound(resp) {
		return nil
	}
	return err
}
