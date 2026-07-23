package api_client

import (
	"encoding/json"
	"fmt"
)

type GitlabGroup struct {
	ID               string                  `json:"id"`
	InstallationID   string                  `json:"installation_id,omitempty"`
	AccountName      string                  `json:"account_name"`
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []ScmPolicyRef          `json:"policies,omitempty"`
	Project          *ScmProjectRef          `json:"project,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
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

// ListGitlabGroups fans out across every GitLab installation so each group
// carries its installation_id (the global /gitlab/integrated_groups/ endpoint
// omits it, which breaks the config-resource for_each workflow).
func (client *APIClient) ListGitlabGroups() ([]GitlabGroup, error) {
	return listScmUnitsByInstallation[GitlabGroup](
		client,
		"/api/shiftleft/gitlab/installations/",
		func(installationID string) string {
			return fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_groups/", installationID)
		},
		func(g *GitlabGroup, installationID string) {
			if g.InstallationID == "" {
				g.InstallationID = installationID
			}
		},
	)
}

// GetGitlabGroup reads via list-filter on the installation-scoped list.
func (client *APIClient) GetGitlabGroup(installationID, groupID string) (*GitlabGroup, error) {
	all, err := getAllScmPages[GitlabGroup](client, fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_groups/", installationID))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == groupID {
			if all[i].InstallationID == "" {
				all[i].InstallationID = installationID
			}
			return &all[i], nil
		}
	}
	return nil, nil // not found -> caller treats nil as drift
}

func (client *APIClient) UpdateGitlabGroup(installationID, groupID string, body ScmInstallationUpdate) (*GitlabGroup, error) {
	// NOTE: the list endpoints are plural ("integrated_groups"), but the update
	// endpoint is singular ("integrated_group"). This mismatch is intentional
	// on the API side, not a typo here.
	if _, err := client.Put(fmt.Sprintf("/api/shiftleft/gitlab/installations/%s/integrated_group/%s/", installationID, groupID), body); err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	return client.GetGitlabGroup(installationID, groupID)
}
