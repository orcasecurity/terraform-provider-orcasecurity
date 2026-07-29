package api_client

import "fmt"

type GithubInstallation struct {
	ID                   string `json:"id"`
	GithubInstallationID int64  `json:"github_installation_id,omitempty"`
	AccountName          string `json:"account_name"`
	GithubAppSettingsURL string `json:"github_app_settings_url,omitempty"`
	ScmUnitCommonFields
}

func (g *GithubInstallation) unitID() string { return g.ID }

func (g *GithubInstallation) stampInstallationID(string) {
	// GitHub installations are top-level units; no parent installation_id to stamp.
}

const githubInstallationsPath = "/api/shiftleft/github/installations/"

func (client *APIClient) ListGithubInstallations() ([]GithubInstallation, error) {
	return getAllScmPages[GithubInstallation](client, githubInstallationsPath, nil)
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
