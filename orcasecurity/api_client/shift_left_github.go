package api_client

import "fmt"

type ScmPolicyRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Builtin bool   `json:"builtin"`
}

// ScmProjectRef is the read-only project reference the API returns on an
// integrated unit that is bound to a scan-all project. Its id is echoed back
// on update as project_id so the association is never dropped.
type ScmProjectRef struct {
	ID string `json:"id"`
}

// PolicyRefIDs returns the ids of the given policy references.
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

// ProjectRefID returns the id of the (possibly nil) project reference.
func ProjectRefID(p *ScmProjectRef) string {
	if p == nil {
		return ""
	}
	return p.ID
}

// ScmInstallationUpdate is the PUT body shared by all SCM providers.
// policies = default_policies ? [] : [ids] (see preprocessAccountToUpdate in UI).
// ProjectID is set instead of policies when the unit is bound to a scan-all
// project, mirroring the UI (project_id XOR policies).
type ScmInstallationUpdate struct {
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []string                `json:"policies"`
	ProjectID        string                  `json:"project_id,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

// ScmUnitCommonFields are the fields every integrated SCM unit (GitHub
// installation, GitLab group, Azure DevOps / Bitbucket account) shares on
// read; the provider-specific DTOs embed it.
type ScmUnitCommonFields struct {
	InstallationMode  string                  `json:"installation_mode,omitempty"`
	DefaultPolicies   bool                    `json:"default_policies"`
	Policies          []ScmPolicyRef          `json:"policies,omitempty"`
	Project           *ScmProjectRef          `json:"project,omitempty"`
	IntegrationStatus string                  `json:"integration_status,omitempty"`
	ConfigSettings    ShiftLeftConfigSettings `json:"configuration_settings"`

	// Read-only status fields (confirmed live on the integrated unit lists).
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

func (g *GithubInstallation) stampInstallationID(string) {
	// No-op: GitHub installations are themselves the units, with no parent
	// installation id to carry (unlike GitLab groups and Azure/Bitbucket
	// accounts, which hang off a parent installation).
}

const githubInstallationsPath = "/api/shiftleft/github/installations/"

func (client *APIClient) ListGithubInstallations() ([]GithubInstallation, error) {
	return getAllScmPages[GithubInstallation](client, githubInstallationsPath)
}

// GetGithubInstallation reads via list-filter; the API defines no single-unit
// GET route for SCM installations (GET on the item path errors — 500 as of
// 2026-07, confirmed live).
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
