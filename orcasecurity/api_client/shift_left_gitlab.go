package api_client

import (
	"encoding/json"
	"fmt"
)

type GitlabGroup struct {
	ID             string `json:"id"`
	InstallationID string `json:"installation_id,omitempty"`
	AccountName    string `json:"account_name"`
	GitlabGroupID  int64  `json:"gitlab_group_id,omitempty"`
	ScmUnitCommonFields
}

func (g *GitlabGroup) unitID() string { return g.ID }

func (g *GitlabGroup) stampInstallationID(id string) {
	if g.InstallationID == "" {
		g.InstallationID = id
	}
}

// UnmarshalJSON maps the GitLab group name into AccountName. Unlike the other
// providers, GitLab's API returns the unit name under `gitlab_group_name`
// (there is no `account_name` field), so fall back to it.
func (g *GitlabGroup) UnmarshalJSON(b []byte) error {
	type alias GitlabGroup
	aux := struct {
		alias
		GitlabGroupName string `json:"gitlab_group_name"`
	}{alias: alias(*g)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	*g = GitlabGroup(aux.alias)
	if g.AccountName == "" {
		g.AccountName = aux.GitlabGroupName
	}
	return nil
}

// GitlabInstallation is a parent GitLab connection (token + server). The
// access token is write-only: the API never echoes it back
// (GitLabInstallationSerializer omits it), so reads carry only its
// name/type metadata.
type GitlabInstallation struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	ServerURL         string `json:"server_url,omitempty"`
	ExternalServerURL string `json:"external_server_url,omitempty"`
	AccessTokenName   string `json:"access_token_name,omitempty"`
	AccessTokenType   string `json:"access_token_type,omitempty"`
	ReadOnly          bool   `json:"read_only"`
	IntegrationStatus string `json:"integration_status,omitempty"`
	CloudIntegration  bool   `json:"cloud_integration"`
}

// GitlabInstallationWrite is the POST/PATCH body. ReadOnly is always sent:
// the API defaults an omitted read_only to false on PATCH (not "unchanged"),
// so partial updates must echo the current value.
type GitlabInstallationWrite struct {
	AccessToken string `json:"access_token,omitempty"`
	Name        string `json:"name,omitempty"`
	ServerURL   string `json:"server_url,omitempty"`
	ReadOnly    bool   `json:"read_only"`
}

func (g *GitlabInstallation) installationID() string { return g.ID }

const gitlabInstallationsPath = "/api/shiftleft/gitlab/installations/"

func (client *APIClient) ListGitlabInstallations() ([]GitlabInstallation, error) {
	return getAllScmPages[GitlabInstallation](client, gitlabInstallationsPath)
}

// GetGitlabInstallation reads via list-filter. Returns nil when absent.
func (client *APIClient) GetGitlabInstallation(id string) (*GitlabInstallation, error) {
	return findScmInstallation[GitlabInstallation](client, gitlabInstallationsPath, id)
}

func (client *APIClient) CreateGitlabInstallation(body GitlabInstallationWrite) (*GitlabInstallation, error) {
	return createScmInstallation[GitlabInstallation](client, gitlabInstallationsPath, body)
}

// UpdateGitlabInstallation PATCHes and re-reads (the PATCH response body is
// empty).
func (client *APIClient) UpdateGitlabInstallation(id string, body GitlabInstallationWrite) (*GitlabInstallation, error) {
	return patchScmInstallationAndReread[GitlabInstallation](client, gitlabInstallationsPath, id, body)
}

func (client *APIClient) DeleteGitlabInstallation(id string) error {
	return deleteScmInstallation(client, gitlabInstallationsPath, id)
}

func gitlabGroupsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_groups/", installationID)
}

// ListGitlabGroups fans out across every GitLab installation so each group
// carries its installation_id (the global /gitlab/integrated_groups/ endpoint
// omits it, which breaks the config-resource for_each workflow).
func (client *APIClient) ListGitlabGroups() ([]GitlabGroup, error) {
	return listScmUnitsByInstallation[GitlabGroup](client, "/api/shiftleft/gitlab/installations/", gitlabGroupsPath)
}

// GetGitlabGroup reads via list-filter on the installation-scoped list by Orca unit UUID.
func (client *APIClient) GetGitlabGroup(installationID, orcaGroupID string) (*GitlabGroup, error) {
	return findScmUnit[GitlabGroup](client, gitlabGroupsPath(installationID), installationID, orcaGroupID)
}

// FindGitlabGroupByGitlabID reads via list-filter matching the GitLab-side numeric group id.
func (client *APIClient) FindGitlabGroupByGitlabID(installationID string, gitlabGroupID int64) (*GitlabGroup, error) {
	all, err := getAllScmPages[GitlabGroup](client, gitlabGroupsPath(installationID))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].GitlabGroupID == gitlabGroupID {
			all[i].stampInstallationID(installationID)
			return &all[i], nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateGitlabGroup(installationID, orcaGroupID string, body ScmInstallationUpdate) (*GitlabGroup, error) {
	// NOTE: the list endpoints are plural ("integrated_groups"), but the update
	// endpoint is singular ("integrated_group"). This mismatch is intentional
	// on the API side, not a typo here.
	updatePath := fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, orcaGroupID)
	return updateScmUnit[GitlabGroup](client, updatePath, gitlabGroupsPath(installationID), installationID, orcaGroupID, body)
}

func (client *APIClient) DeleteGitlabGroup(installationID, orcaGroupID string) error {
	return deleteScmPathIgnoring404(client,
		fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, orcaGroupID))
}

// GitlabUnitIntegrate is the scan-all (empty repos) create body for a GitLab group.
type GitlabUnitIntegrate struct {
	InstallationID string
	GitlabGroupID  int64
	Body           ScmInstallationUpdate
}

func (client *APIClient) IntegrateGitlabUnit(req GitlabUnitIntegrate) error {
	body := struct {
		InstallationID        string                  `json:"installation_id"`
		GroupID               int64                   `json:"group_id"`
		InstallationMode      string                  `json:"installation_mode,omitempty"`
		DefaultPolicies       bool                    `json:"default_policies"`
		Policies              []string                `json:"policies"`
		ProjectID             string                  `json:"project_id,omitempty"`
		ConfigurationSettings ShiftLeftConfigSettings `json:"configuration_settings"`
		Repositories          []struct{}              `json:"repositories"`
	}{
		InstallationID:        req.InstallationID,
		GroupID:               req.GitlabGroupID,
		InstallationMode:      req.Body.InstallationMode,
		DefaultPolicies:       req.Body.DefaultPolicies,
		Policies:              req.Body.Policies,
		ProjectID:             req.Body.ProjectID,
		ConfigurationSettings: req.Body.ConfigSettings,
		Repositories:          []struct{}{},
	}
	if req.Body.ProjectID != "" {
		body.Policies = nil
	}
	return client.integrateScmRepositories("gitlab", body)
}
