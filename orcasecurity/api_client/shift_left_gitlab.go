package api_client

import (
	"encoding/json"
	"fmt"
)

type GitlabGroup struct {
	ID                string                  `json:"id"`
	InstallationID    string                  `json:"installation_id,omitempty"`
	AccountName       string                  `json:"account_name"`
	InstallationMode  string                  `json:"installation_mode,omitempty"`
	DefaultPolicies   bool                    `json:"default_policies"`
	Policies          []ScmPolicyRef          `json:"policies,omitempty"`
	Project           *ScmProjectRef          `json:"project,omitempty"`
	IntegrationStatus string                  `json:"integration_status,omitempty"`
	ConfigSettings    ShiftLeftConfigSettings `json:"configuration_settings"`
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

func gitlabGroupsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_groups/", installationID)
}

// ListGitlabGroups fans out across every GitLab installation so each group
// carries its installation_id (the global /gitlab/integrated_groups/ endpoint
// omits it, which breaks the config-resource for_each workflow).
func (client *APIClient) ListGitlabGroups() ([]GitlabGroup, error) {
	return listScmUnitsByInstallation[GitlabGroup](client, "/api/shiftleft/gitlab/installations/", gitlabGroupsPath)
}

// GetGitlabGroup reads via list-filter on the installation-scoped list.
func (client *APIClient) GetGitlabGroup(installationID, groupID string) (*GitlabGroup, error) {
	return findScmUnit[GitlabGroup](client, gitlabGroupsPath(installationID), installationID, groupID)
}

func (client *APIClient) UpdateGitlabGroup(installationID, groupID string, body ScmInstallationUpdate) (*GitlabGroup, error) {
	// NOTE: the list endpoints are plural ("integrated_groups"), but the update
	// endpoint is singular ("integrated_group"). This mismatch is intentional
	// on the API side, not a typo here.
	updatePath := fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, groupID)
	return updateScmUnit[GitlabGroup](client, updatePath, gitlabGroupsPath(installationID), installationID, groupID, body)
}
