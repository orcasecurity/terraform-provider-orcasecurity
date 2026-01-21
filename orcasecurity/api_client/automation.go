package api_client

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FlexibleString string

// UnmarshalJSON implements the json.Unmarshaler interface
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*fs = FlexibleString(str)
		return nil
	}

	// If that fails, try to unmarshal as a number (float64 to handle both int and float)
	var num float64
	if err := json.Unmarshal(data, &num); err == nil {
		if num == float64(int64(num)) {
			*fs = FlexibleString(strconv.FormatInt(int64(num), 10))
		} else {
			*fs = FlexibleString(strconv.FormatFloat(num, 'f', -1, 64))
		}
		return nil
	}

	return fmt.Errorf("FlexibleString: cannot unmarshal %s into string or number", string(data))
}

// MarshalJSON implements the json.Marshaler interface
func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(fs))
}

// String returns the string value
func (fs FlexibleString) String() string {
	return string(fs)
}

const AutomationAlertDismissalID = 1
const AutomationAlertScoreChangeID = 28
const AutomationSnoozeID = 31

// Communication & Notification Actions
const AutomationSlackID = 2 // Covers both Slack v1 and Slack v2 integrations
const AutomationPagerDutyID = 3
const AutomationOpsgenieID = 4
const AutomationEmailID = 5
const AutomationMsTeamsID = 19

// SIEM & Security Tools
const AutomationSumoLogicID = 6
const AutomationAzureSentinelID = 7
const AutomationSplunkID = 8
const AutomationAWSSecurityHubID = 37
const AutomationChronicleID = 27
const AutomationSiemID = 21

// Ticketing & Project Management
const AutomationJiraID = 10
const AutomationJiraServerID = 23
const AutomationServiceNowIncidentsID = 22
const AutomationServiceNowSIIncidentsID = 24
const AutomationMondayID = 38
const AutomationLinearID = 40

// Cloud & Infrastructure
const AutomationGcpPubSubID = 13
const AutomationAwsSqsID = 33
const AutomationAwsSnsID = 34
const AutomationAwsSecurityLakeID = 25
const AutomationAzureDevopsID = 17

// Data & Analytics
const AutomationSnowflakeID = 26
const AutomationCoralogixID = 36
const AutomationDatadogID = 18
const AutomationCriblID = 29

// Automation & Orchestration
const AutomationWebhookID = 12
const AutomationTinesID = 30
const AutomationTorqID = 16

// Security & CDN
const AutomationCloudflareID = 32
const AutomationAkamaiID = 39
const AutomationPantherID = 41

// Remediation & Custom
const AutomationRemediationID = 20
const AutomationOpusID = 35

// Deprecated/Legacy
const AutomationGoogleSecurityOperationsSIEMID = 27 // Same as ChronicleID

type AutomationRange struct {
	Gte *FlexibleString `json:"gte,omitempty"`
	Lte *FlexibleString `json:"lte,omitempty"`
	Gt  *FlexibleString `json:"gt,omitempty"`
	Lt  *FlexibleString `json:"lt,omitempty"`
	Eq  *FlexibleString `json:"eq,omitempty"`
}

type AutomationFilter struct {
	Field         string           `json:"field"`
	Includes      []string         `json:"includes,omitempty"`
	Excludes      []string         `json:"excludes,omitempty"`
	Prefix        []string         `json:"prefix,omitempty"`
	ExcludePrefix []string         `json:"exclude_prefix,omitempty"`
	Range         *AutomationRange `json:"range,omitempty"`
}

type AutomationQuery struct {
	Filter []AutomationFilter `json:"filter"`
}

type AutomationAction struct {
	ID             string                 `json:"id,omitempty"`
	Type           int32                  `json:"type"`
	OrganizationID string                 `json:"organization,omitempty"`
	Data           map[string]interface{} `json:"data"`
}

func (a *AutomationAction) IsAlertDismissalTemplate() bool {
	return a.Type == AutomationAlertDismissalID
}

func (a *AutomationAction) IsAzureDevopsTemplate() bool {
	return a.Type == AutomationAzureDevopsID
}

func (a *AutomationAction) IsEmailTemplate() bool {
	return a.Type == AutomationEmailID
}

func (a *AutomationAction) IsJiraTemplate() bool {
	return a.Type == AutomationJiraID
}

func (a *AutomationAction) IsPagerDutyTemplate() bool {
	return a.Type == AutomationPagerDutyID
}

func (a *AutomationAction) IsSumoLogicTemplate() bool {
	return a.Type == AutomationSumoLogicID
}

func (a *AutomationAction) IsWebhookTemplate() bool {
	return a.Type == AutomationWebhookID
}

func (a *AutomationAction) IsSlackV2() bool {
	return a.Type == AutomationSlackID
}

type Automation struct {
	ID             string             `json:"id,omitempty"`
	Name           string             `json:"name"`
	BusinessUnits  []string           `json:"business_units"`
	Enabled        bool               `json:"is_enabled"`
	SonarQuery     map[string]int     `json:"sonar_query"`
	Description    string             `json:"description"`
	OrganizationID string             `json:"organization,omitempty"`
	Query          AutomationQuery    `json:"dsl_filter"`
	Actions        []AutomationAction `json:"actions"`
}

type automationAPIResponseType struct {
	Data Automation `json:"data"`
}

func (client *APIClient) GetAutomation(automationID string) (*Automation, error) {
	resp, err := client.Get(fmt.Sprintf("/api/rules/%s", automationID))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DoesAutomationExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rules/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) CreateAutomation(automation Automation) (*Automation, error) {
	resp, err := client.Post("/api/rules", automation)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil

}

func (client *APIClient) UpdateAutomation(ID string, data Automation) (*Automation, error) {
	resp, err := client.Put(fmt.Sprintf("/api/rules/%s", ID), data)
	if err != nil {
		return nil, err
	}

	response := automationAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteAutomation(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/rules/%s", ID))
	return err
}
