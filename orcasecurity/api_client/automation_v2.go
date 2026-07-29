package api_client

import (
	"fmt"
)

type AutomationV2 struct {
	ID             string               `json:"id,omitempty"`
	Name           string               `json:"name"`
	BusinessUnits  []string             `json:"business_units"`
	Description    string               `json:"description"`
	Status         string               `json:"status"`
	Filter         AutomationV2Filter   `json:"filter"`
	Actions        []AutomationV2Action `json:"actions"`
	OrganizationID string               `json:"organization,omitempty"`
	EndTime        string               `json:"end_time,omitempty"`
	CreatedAt      string               `json:"created_at,omitempty"`
	UpdatedAt      string               `json:"updated_at,omitempty"`
	// Priority is the global 1-based evaluation order (nominally dense, but
	// legacy data may contain gaps or duplicates). Read-only on the automation
	// CRUD endpoints; writable only via SetAutomationV2Priority.
	Priority *int64 `json:"priority,omitempty"`
}

type AutomationV2Filter struct {
	SonarQuery AutomationV2SonarQuery `json:"sonar_query"`
}

type AutomationV2SonarQuery struct {
	Models []string               `json:"models"`
	Type   string                 `json:"type"`
	With   map[string]interface{} `json:"with,omitempty"` // Complex nested filter structure
}

type AutomationV2Action struct {
	ID             string                 `json:"id,omitempty"`
	Type           int32                  `json:"type"`
	Data           map[string]interface{} `json:"data"`
	ExternalConfig *string                `json:"external_config,omitempty"`
	// SiemToken carries the config-reference UUID for the SIEM "API Token" action
	// (type AutomationSiemID), which uses siem_token instead of external_config.
	SiemToken *string `json:"siem_token,omitempty"`
}

func (client *APIClient) GetAutomationV2(automationID string) (*AutomationV2, error) {
	resp, err := client.Get(fmt.Sprintf("/api/automations/%s", automationID))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	return readData[AutomationV2](resp)
}

// ListAutomationsV2 returns every automation in the organization in server
// evaluation order (priority ascending, creation time as tiebreak), paging
// through the list endpoint. Priorities in real-world data may contain
// duplicates or gaps from legacy rows; the server order is still
// deterministic.
func (client *APIClient) ListAutomationsV2() ([]AutomationV2, error) {
	const pageLimit = 300
	const maxPages = 500
	return paginateOffset[AutomationV2](client, "/api/automations", pageLimit, maxPages)
}

func (client *APIClient) DoesAutomationV2Exist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/automations/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateAutomationV2(automation AutomationV2, applyOnExisting bool) (*AutomationV2, error) {
	path := "/api/automations"
	if applyOnExisting {
		path += "?apply_on_existing=true"
	}
	resp, err := client.Post(path, automation)
	if err != nil {
		return nil, err
	}

	return readData[AutomationV2](resp)
}

func (client *APIClient) UpdateAutomationV2(ID string, data AutomationV2) (*AutomationV2, error) {
	resp, err := client.Put(fmt.Sprintf("/api/automations/%s", ID), data)
	if err != nil {
		return nil, err
	}

	return readData[AutomationV2](resp)
}

// SetAutomationV2Priority moves the automation to the given evaluation-order
// position via the dedicated priority endpoint. The server renumbers displaced
// automations atomically and silently clamps values above the organization's
// current highest priority (Least(new, Max(priority)), NOT the automation
// count — legacy data can have gaps and duplicates), so callers must compare
// the returned Priority with the requested one.
func (client *APIClient) SetAutomationV2Priority(automationID string, priority int64) (*AutomationV2, error) {
	payload := struct {
		Priority int64 `json:"priority"`
	}{Priority: priority}

	resp, err := client.Put(fmt.Sprintf("/api/automations/%s/priority", automationID), payload)
	if err != nil {
		return nil, err
	}

	return readData[AutomationV2](resp)
}

func (client *APIClient) DeleteAutomationV2(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/automations/%s", ID))
	return err
}
