package api_client

import "fmt"

type ScmPolicyRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Builtin bool   `json:"builtin"`
}

// ScmInstallationUpdate is the PUT body shared by all SCM providers.
// policies = default_policies ? [] : [ids] (see preprocessAccountToUpdate in UI).
type ScmInstallationUpdate struct {
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []string                `json:"policies"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

type GithubInstallation struct {
	ID                   string                  `json:"id"`
	GithubInstallationID int64                   `json:"github_installation_id,omitempty"`
	AccountName          string                  `json:"account_name"`
	InstallationMode     string                  `json:"installation_mode,omitempty"`
	DefaultPolicies      bool                    `json:"default_policies"`
	Policies             []ScmPolicyRef          `json:"policies,omitempty"`
	ConfigSettings       ShiftLeftConfigSettings `json:"configuration_settings"`
}

func (client *APIClient) ListGithubInstallations() ([]GithubInstallation, error) {
	return getAllScmPages[GithubInstallation](client, "/api/shiftleft/github/installations/")
}

// GetGithubInstallation reads via list-filter; single-GET returns 500.
func (client *APIClient) GetGithubInstallation(id string) (*GithubInstallation, error) {
	all, err := client.ListGithubInstallations()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil // not found -> caller treats nil as drift
}

func (client *APIClient) UpdateGithubInstallation(id string, body ScmInstallationUpdate) (*GithubInstallation, error) {
	if _, err := client.Put(fmt.Sprintf("/api/shiftleft/github/installations/%s/", id), body); err != nil {
		return nil, err
	}
	return client.GetGithubInstallation(id)
}
