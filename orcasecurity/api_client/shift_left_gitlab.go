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

// GitLab returns gitlab_group_name, not account_name.
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

// Omitted read_only defaults to false on PATCH, not "unchanged".
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

func (client *APIClient) GetGitlabInstallation(id string) (*GitlabInstallation, error) {
	return findScmInstallation[GitlabInstallation](client, gitlabInstallationsPath, id)
}

func (client *APIClient) CreateGitlabInstallation(body GitlabInstallationWrite) (*GitlabInstallation, error) {
	return createScmInstallation[GitlabInstallation](client, gitlabInstallationsPath, body)
}

func (client *APIClient) UpdateGitlabInstallation(id string, body GitlabInstallationWrite) (*GitlabInstallation, error) {
	return patchScmInstallationAndReread[GitlabInstallation](client, gitlabInstallationsPath, id, body)
}

func (client *APIClient) DeleteGitlabInstallation(id string) error {
	return deleteScmInstallation(client, gitlabInstallationsPath, id)
}

func gitlabGroupsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_groups/", installationID)
}

func (client *APIClient) ListGitlabGroups() ([]GitlabGroup, error) {
	return listScmUnitsByInstallation[GitlabGroup](client, "/api/shiftleft/gitlab/installations/", gitlabGroupsPath)
}

func (client *APIClient) GetGitlabGroup(installationID, orcaGroupID string) (*GitlabGroup, error) {
	return findScmUnit[GitlabGroup](client, gitlabGroupsPath(installationID), installationID, orcaGroupID)
}

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
	// List path is integrated_groups; update path is integrated_group.
	updatePath := fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, orcaGroupID)
	return updateScmUnit[GitlabGroup](client, updatePath, gitlabGroupsPath(installationID), installationID, orcaGroupID, body)
}

func (client *APIClient) DeleteGitlabGroup(installationID, orcaGroupID string) error {
	return deleteScmPathIgnoring404(client,
		fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, orcaGroupID))
}

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
