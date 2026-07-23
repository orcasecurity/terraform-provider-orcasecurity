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

type GithubInstallation struct {
	ID                   string                  `json:"id"`
	GithubInstallationID int64                   `json:"github_installation_id,omitempty"`
	AccountName          string                  `json:"account_name"`
	InstallationMode     string                  `json:"installation_mode,omitempty"`
	DefaultPolicies      bool                    `json:"default_policies"`
	Policies             []ScmPolicyRef          `json:"policies,omitempty"`
	Project              *ScmProjectRef          `json:"project,omitempty"`
	IntegrationStatus    string                  `json:"integration_status,omitempty"`
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
	client.invalidateScmListCache()
	return client.GetGithubInstallation(id)
}
