package api_client

import "fmt"

type ScmPolicyRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Builtin bool   `json:"builtin"`
}

type ScmProjectRef struct {
	ID string `json:"id"`
}

func PolicyRefIDs(refs []ScmPolicyRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, r := range refs {
		ids = append(ids, r.ID)
	}
	return ids
}

func ProjectRefID(p *ScmProjectRef) string {
	if p == nil {
		return ""
	}
	return p.ID
}

// PUT sends project_id XOR policies.
type ScmInstallationUpdate struct {
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []string                `json:"policies"`
	ProjectID        string                  `json:"project_id,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

type ScmUnitCommonFields struct {
	InstallationMode  string                  `json:"installation_mode,omitempty"`
	DefaultPolicies   bool                    `json:"default_policies"`
	Policies          []ScmPolicyRef          `json:"policies,omitempty"`
	Project           *ScmProjectRef          `json:"project,omitempty"`
	IntegrationStatus string                  `json:"integration_status,omitempty"`
	ConfigSettings    ShiftLeftConfigSettings `json:"configuration_settings"`

	ScanAllState                string `json:"scan_all_state,omitempty"`
	IntegratedRepositoriesCount int64  `json:"integrated_repositories_count,omitempty"`
	ScmPosturePolicyID          string `json:"scm_posture_policy_id,omitempty"`
}

type GithubInstallation struct {
	ID                   string `json:"id"`
	GithubInstallationID int64  `json:"github_installation_id,omitempty"`
	AccountName          string `json:"account_name"`
	GithubAppSettingsURL string `json:"github_app_settings_url,omitempty"`
	ScmUnitCommonFields
}

func (g *GithubInstallation) unitID() string { return g.ID }

func (g *GithubInstallation) stampInstallationID(string) {}

const githubInstallationsPath = "/api/shiftleft/github/installations/"

func (client *APIClient) ListGithubInstallations() ([]GithubInstallation, error) {
	return getAllScmPages[GithubInstallation](client, githubInstallationsPath)
}

// No single-unit GET route; list-filter only.
func (client *APIClient) GetGithubInstallation(id string) (*GithubInstallation, error) {
	return findScmUnit[GithubInstallation](client, githubInstallationsPath, "", id)
}

func (client *APIClient) UpdateGithubInstallation(id string, body ScmInstallationUpdate) (*GithubInstallation, error) {
	updatePath := fmt.Sprintf("%s%s/", githubInstallationsPath, id)
	return updateScmUnit[GithubInstallation](client, updatePath, githubInstallationsPath, "", id, body)
}

func (client *APIClient) DeleteGithubInstallation(id string) error {
	return deleteScmPathIgnoring404(client, fmt.Sprintf("%s%s/", githubInstallationsPath, id))
}
