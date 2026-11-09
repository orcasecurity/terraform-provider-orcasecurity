package api_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type CustomSonarAlertRemediationText struct {
	AlertType string `json:"alert_type"`
	Enable    bool   `json:"enabled"`
	Text      string `json:"custom_text"`
}

type CustomAlert struct {
	ID                   string                                `json:"rule_id,omitempty"`
	OrganizationID       string                                `json:"organization,omitempty"`
	Category             string                                `json:"category"`
	ContextScore         bool                                  `json:"context_score"`
	OrcaScore            float64                               `json:"orca_score"`
	Name                 string                                `json:"name"`
	Description          string                                `json:"details"`
	Rule                 string                                `json:"rule"`
	RuleType             string                                `json:"rule_type,omitempty"` // also "alert_type"
	Enabled              bool                                  `json:"enabled"`
	ComplianceFrameworks []CustomSonarAlertComplianceFramework `json:"compliance_frameworks,omitempty"`
	RemediationText      *CustomSonarAlertRemediationText      // managed in a separate API call
}

func (client *APIClient) GetCustomSonarAlert(id string) (*CustomAlert, error) {
	type responseType struct {
		Data CustomAlert `json:"data"`
	}
	resp, missing, err := client.getSonarRule(id)
	if missing {
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
	remediation, err := client.GetCustomSonarAlertRemediationText(alert.RuleType)
	if err != nil {
		return nil, fmt.Errorf("error fetching remediation text: %s", err.Error())
	}
	alert.RemediationText = &CustomSonarAlertRemediationText{
		AlertType: remediation.AlertType,
		Enable:    remediation.Enable,
		Text:      remediation.Text,
	}

	return &alert, nil
}

func (client *APIClient) CreateCustomSonarAlert(data CustomAlert) (*CustomAlert, error) {
	type responseType struct {
		Data CustomAlert `json:"data"`
	}

	resp, err := client.Post("/api/sonar/rules", data)
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	alert := &response.Data

	// set remediation
	if data.RemediationText != nil {
		data.RemediationText.AlertType = alert.RuleType
		if err = client.SetCustomSonarAlertRemediationText(*data.RemediationText); err != nil {
			return nil, fmt.Errorf("remediation text create failed: %s", err.Error())
		}
	}

	return alert, nil
}

func (client *APIClient) UpdateCustomSonarAlert(id string, data CustomAlert) (*CustomAlert, error) {
	instance, err := client.UpdateCustomSonarAlertRule(id, data)
	if err != nil {
		return nil, err
	}
	if err := client.UpdateCustomSonarAlertRemediation(data); err != nil {
		return nil, err
	}
	return instance, nil
}

// UpdateCustomSonarAlertRule writes the rule alone. Remediation text lives behind its own
// endpoint, and a rule write is repeated when compliance framework links are
// replaced, so the two are separate: a remediation failure must not be reported
// as a failed rule write, and remediation must not be written twice.
func (client *APIClient) UpdateCustomSonarAlertRule(id string, data CustomAlert) (*CustomAlert, error) {
	resp, err := client.Put(fmt.Sprintf("/api/sonar/rules/%s", id), data)
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data CustomAlert `json:"data"`
	}

	response := responseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// UpdateCustomSonarAlertRemediation brings the alert's remediation text in line with the plan.
func (client *APIClient) UpdateCustomSonarAlertRemediation(data CustomAlert) error {
	if data.RemediationText == nil {
		if err := client.DeleteCustomSonarAlertRemediationText(CustomSonarAlertRemediationText{AlertType: data.RuleType}); err != nil {
			return fmt.Errorf("remediation text delete failed: %s", err.Error())
		}
		return nil
	}
	if err := client.SetCustomSonarAlertRemediationText(*data.RemediationText); err != nil {
		return fmt.Errorf("remediation text update failed: %s", err.Error())
	}
	return nil
}
func (client *APIClient) DeleteCustomSonarAlert(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/sonar/rules/%s", id))
	return err
}

func (client *APIClient) GetCustomSonarAlertRemediationText(ruleType string) (*CustomSonarAlertRemediationText, error) {
	resp, err := client.Get(fmt.Sprintf("/api/alerts/custom_remediation_text?alert_type=%s", ruleType))
	if resp != nil && resp.StatusCode() == 404 {
		return &CustomSonarAlertRemediationText{}, nil
	}

	if err != nil {
		return nil, err
	}

	response := CustomSonarAlertRemediationText{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}
	return &response, err
}

func (client *APIClient) SetCustomSonarAlertRemediationText(data CustomSonarAlertRemediationText) error {
	resp, err := client.Put("/api/alerts/custom_remediation_text", data)
	if resp != nil && resp.StatusCode() == 404 {
		_, err = client.Post("/api/alerts/custom_remediation_text", data)
	}
	return err
}
func (client *APIClient) DeleteCustomSonarAlertRemediationText(data CustomSonarAlertRemediationText) error {
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

	resp, err := client.roundTripWithRetry(*req)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 404 {
		return nil
	}

	if !resp.IsOk() {
		return fmt.Errorf("operation failed")
	}

	return nil
}

func (client *APIClient) GetSonarAlertCategories() ([]string, error) {
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
