package api_client

// AlertComplianceFramework is one `compliance_frameworks` entry on a custom
// alert. Discovery and sonar alerts send and receive the same payload, so they
// share the type; the per-resource names stay as aliases because that is how the
// API documents them.
type AlertComplianceFramework struct {
	Name           string `json:"compliance_framework"`
	Category       string `json:"category"`
	SubCategory    string `json:"sub_category,omitempty"`
	SubSubCategory string `json:"sub_sub_category,omitempty"`
	Priority       string `json:"priority"`
}

type CustomDiscoveryAlertComplianceFramework = AlertComplianceFramework

type CustomSonarAlertComplianceFramework = AlertComplianceFramework
