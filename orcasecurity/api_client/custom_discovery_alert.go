package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type customDiscoveryAlertAPIResponseType struct {
	Data CustomDiscoveryAlert `json:"data"`
}

type CustomDiscoveryAlert struct {
	ID                   string                                    `json:"rule_id,omitempty"`
	Name                 string                                    `json:"name"`
	Description          string                                    `json:"details"`
	Negation             string                                    `json:"negation"`
	OrganizationID       string                                    `json:"organization,omitempty"`
	Category             string                                    `json:"category"`
	ContextScore         bool                                      `json:"context_score"`
	OrcaScore            float64                                   `json:"orca_score"`
	RuleJson             map[string]interface{}                    `json:"rule_json"`
	RuleType             string                                    `json:"rule_type,omitempty"`
	ComplianceFrameworks []CustomDiscoveryAlertComplianceFramework `json:"compliance_frameworks,omitempty"`
	RemediationText      *CustomDiscoveryAlertRemediationText      // managed in a separate API call
}

type CustomDiscoveryAlertComplianceFramework struct {
	Name     string `json:"compliance_framework"`
	Section  string `json:"category"`
	Priority string `json:"priority"`
}

type CustomDiscoveryAlertRemediationText struct {
	AlertType string `json:"alert_type"`
	Enable    bool   `json:"enabled"`
	Text      string `json:"custom_text"`
}

func (client *APIClient) DoesCustomDiscoveryAlertExist(id string) (bool, error) {
	resp, err := client.Head(fmt.Sprintf("/api/sonar/rules/%s", id))
	if resp.StatusCode() == 404 || resp.StatusCode() == 500 {
		return false, nil
	}

	// some other error
	if err != nil {
		return false, err
	}
	return true, nil
}

func (client *APIClient) GetCustomDiscoveryAlert(id string) (*CustomDiscoveryAlert, error) {
	type responseType struct {
		Data CustomDiscoveryAlert `json:"data"`
	}
	resp, err := client.Get(fmt.Sprintf("/api/sonar/rules/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	alert := response.Data

	// add remediation value
	remediation, err := client.GetCustomDiscoveryAlertRemediationText(alert.RuleType)
	if err != nil {
		return nil, fmt.Errorf("error fetching remediation text: %s", err.Error())
	}
	alert.RemediationText = &CustomDiscoveryAlertRemediationText{
		AlertType: remediation.AlertType,
		Enable:    remediation.Enable,
		Text:      remediation.Text,
	}

	return &alert, nil
}

func (client *APIClient) CreateCustomDiscoveryAlert(data CustomDiscoveryAlert) (*CustomDiscoveryAlert, error) {
	resp, err := client.Post("/api/sonar/rules", data)
	if err != nil {
		return nil, err
	}

	response := customDiscoveryAlertAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	alert := &response.Data

	// set remediation
	if data.RemediationText != nil {
		data.RemediationText.AlertType = alert.RuleType
		if err = client.SetCustomRemediationText(*data.RemediationText); err != nil {
			return nil, fmt.Errorf("remediation text create failed: %s", err.Error())
		}
	}

	return alert, nil
}

func (client *APIClient) UpdateCustomDiscoveryAlert(id string, data CustomDiscoveryAlert) (*CustomDiscoveryAlert, error) {
	resp, err := client.Put(fmt.Sprintf("/api/sonar/rules/%s", id), data)
	if err != nil {
		return nil, err
	}

	response := customDiscoveryAlertAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	// update remediation
	if data.RemediationText == nil {
		if err = client.DeleteCustomRemediationText(CustomDiscoveryAlertRemediationText{
			AlertType: data.RuleType,
		}); err != nil {
			return nil, fmt.Errorf("remediation text delete failed: %s", err.Error())
		}
	} else {
		if err = client.SetCustomRemediationText(*data.RemediationText); err != nil {
			return nil, fmt.Errorf("remediation text update failed: %s", err.Error())
		}
	}

	return &response.Data, err
}
func (client *APIClient) DeleteCustomDiscoveryAlert(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/sonar/rules/%s", id))
	return err
}

func (client *APIClient) GetCustomDiscoveryAlertRemediationText(ruleType string) (*CustomDiscoveryAlertRemediationText, error) {
	resp, err := client.Get(fmt.Sprintf("/api/alerts/custom_remediation_text?alert_type=%s", ruleType))
	if resp.StatusCode() == 404 {
		return &CustomDiscoveryAlertRemediationText{}, nil
	}

	if err != nil {
		return nil, err
	}

	response := CustomDiscoveryAlertRemediationText{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response, err
}

func (client *APIClient) SetCustomRemediationText(data CustomDiscoveryAlertRemediationText) error {
	resp, err := client.Put("/api/alerts/custom_remediation_text", data)
	if resp.StatusCode() == 404 {
		_, err = client.Post("/api/alerts/custom_remediation_text", data)
	}
	return err
}
func (client *APIClient) DeleteCustomRemediationText(data CustomDiscoveryAlertRemediationText) error {
	payload, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/alerts/custom_remediation_text", client.APIEndpoint),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return err
	}

	resp, err := client.Execute(*req)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("operation failed")
	}

	return nil
}

func (client *APIClient) GetAlertCategories() ([]string, error) {
	type responseType struct {
		Data []string `json:"data"`
	}
	res, err := client.Get("/api/alerts/catalog/category")
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = res.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
