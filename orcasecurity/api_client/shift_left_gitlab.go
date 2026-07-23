package api_client

import "fmt"

type GitlabGroup struct {
	ID               string                  `json:"id"`
	InstallationID   string                  `json:"installation_id,omitempty"`
	AccountName      string                  `json:"account_name"`
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []ScmPolicyRef          `json:"policies,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

func (client *APIClient) ListGitlabGroups() ([]GitlabGroup, error) {
	return getAllScmPages[GitlabGroup](client, "/api/shiftleft/gitlab/integrated_groups/")
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
	return client.GetGitlabGroup(installationID, groupID)
}
