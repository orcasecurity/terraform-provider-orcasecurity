package api_client

import (
	"fmt"
	"net/http"
)

const (
	complianceFrameworkSelectPath  = "/api/compliance/frameworks/select"
	complianceFrameworkCatalogPath = "/api/compliance/catalog"
)

// ComplianceFramework is one entry from GET /api/compliance/frameworks/select
// and the data envelope of GET /api/compliance/frameworks/{id}. Optional keys
// differ between system and custom frameworks; missing values stay nil.
type ComplianceFramework struct {
	ID                         string   `json:"id"`
	Active                     bool     `json:"active"`
	SelectionScopes            []string `json:"selection_scopes"`
	DisplayName                string   `json:"display_name"`
	Custom                     bool     `json:"custom"`
	Description                *string  `json:"description"`
	Type                       *string  `json:"type"`
	Version                    *string  `json:"version"`
	VersionAgnosticDisplayName *string  `json:"version_agnostic_display_name"`
	IsReady                    *bool    `json:"is_ready"`
	FrameworkCloudVendors      []string `json:"framework_cloud_vendors"`
	IconFamily                 *string  `json:"icon_family"`
	OrcaEndOfSupportDate       *string  `json:"orca_end_of_support_date"`
	Visibility                 *string  `json:"visibility"`
	OriginType                 *string  `json:"origin_type"`
	CreatedAt                  *string  `json:"created_at"`
	UpdatedAt                  *string  `json:"updated_at"`
	CreatedBy                  *string  `json:"created_by"`
	UpdatedBy                  *string  `json:"updated_by"`
	IsForcedCloudVendors       *bool    `json:"is_forced_cloud_vendors"`
}

type complianceFrameworkReadAPIResponse struct {
	Data ComplianceFramework `json:"data"`
}

type complianceFrameworkSelectRequest struct {
	FrameworkIDs []string `json:"framework_ids"`
	Scope        string   `json:"scope"`
}

// ComplianceCatalogTest is one control inside GET /api/compliance/catalog/{id}.
type ComplianceCatalogTest struct {
	Name              string   `json:"name"`
	RuleID            string   `json:"rule_id"`
	ReferenceID       string   `json:"reference_id"`
	OriginFrameworkID string   `json:"origin_framework_id"`
	CloudVendors      []string `json:"cloud_vendors"`
	ControlUniqueID   string   `json:"control_unique_id"`
	Priority          string   `json:"priority"`
}

// ComplianceCatalogSection is one node of the catalog section tree. The
// nested `sections` key is absent on a leaf, not empty.
type ComplianceCatalogSection struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	TotalTests int                        `json:"total_tests"`
	Tests      []ComplianceCatalogTest    `json:"tests"`
	Sections   []ComplianceCatalogSection `json:"sections"`
}

// ComplianceCatalogFramework is the catalog view of one framework, including
// the nested section/test tree that GET /api/compliance/frameworks/{id} omits.
type ComplianceCatalogFramework struct {
	Name                 string                     `json:"name"`
	DisplayName          string                     `json:"display_name"`
	FrameworkID          string                     `json:"framework_id"`
	Custom               bool                       `json:"custom"`
	Version              string                     `json:"version"`
	Type                 *string                    `json:"type"`
	TotalSections        int                        `json:"total_sections"`
	CloudVendors         []string                   `json:"cloud_vendors"`
	IsForcedCloudVendors bool                       `json:"is_forced_cloud_vendors"`
	Visibility           *string                    `json:"visibility"`
	IconFamily           *string                    `json:"icon_family"`
	OrcaEndOfSupportDate *string                    `json:"orca_end_of_support_date"`
	Sections             []ComplianceCatalogSection `json:"sections"`
}

type complianceCatalogAPIResponse struct {
	Data struct {
		Frameworks      []ComplianceCatalogFramework `json:"frameworks"`
		TotalFrameworks int                          `json:"total_frameworks"`
	} `json:"data"`
}

func isNotFound(resp *APIResponse) bool {
	return resp != nil && resp.StatusCode() == http.StatusNotFound
}

// GetComplianceFrameworkSelections returns the tenant's select map, keyed by
// framework id. The payload is a bare JSON object, not a list or envelope.
func (client *APIClient) GetComplianceFrameworkSelections() (map[string]ComplianceFramework, error) {
	resp, err := client.Get(complianceFrameworkSelectPath)
	if err != nil {
		return nil, err
	}
	var out map[string]ComplianceFramework
	if err = resp.ReadJSON(&out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]ComplianceFramework{}
	}
	for id, fw := range out {
		if fw.ID == "" {
			fw.ID = id
			out[id] = fw
		}
	}
	return out, nil
}

// SelectComplianceFrameworks POSTs framework_ids onto the given scope
// (`user` or `organization`). The call is idempotent for a scope already held.
func (client *APIClient) SelectComplianceFrameworks(ids []string, scope string) error {
	_, err := client.Post(complianceFrameworkSelectPath, complianceFrameworkSelectRequest{
		FrameworkIDs: ids,
		Scope:        scope,
	})
	return err
}

// DeselectComplianceFrameworks DELETEs framework_ids from the given scope.
// The endpoint requires a JSON body. Idempotent for a scope not held.
func (client *APIClient) DeselectComplianceFrameworks(ids []string, scope string) error {
	_, err := client.DeleteWithBody(complianceFrameworkSelectPath, complianceFrameworkSelectRequest{
		FrameworkIDs: ids,
		Scope:        scope,
	})
	return err
}

// GetComplianceFramework returns one framework's metadata.
// 404 (missing or tenant-invisible) is (nil, nil); every other non-2xx is an error.
func (client *APIClient) GetComplianceFramework(id string) (*ComplianceFramework, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", customComplianceFrameworkBasePath, id))
	if isNotFound(resp) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response := complianceFrameworkReadAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	if response.Data.ID == "" {
		response.Data.ID = id
	}
	return &response.Data, nil
}

// GetComplianceCatalogFramework returns the catalog tree for one framework.
// 404 or an empty frameworks list is (nil, nil).
func (client *APIClient) GetComplianceCatalogFramework(id string) (*ComplianceCatalogFramework, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", complianceFrameworkCatalogPath, id))
	if isNotFound(resp) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response := complianceCatalogAPIResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	if len(response.Data.Frameworks) == 0 {
		return nil, nil
	}
	return &response.Data.Frameworks[0], nil
}
