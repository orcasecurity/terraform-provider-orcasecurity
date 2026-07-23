package api_client

import "fmt"

type BitbucketAccount struct {
	ID             string `json:"id"`
	InstallationID string `json:"installation_id,omitempty"`
	AccountName    string `json:"account_name"`
	ScmUnitCommonFields
}

func (a *BitbucketAccount) unitID() string { return a.ID }

func (a *BitbucketAccount) stampInstallationID(id string) {
	if a.InstallationID == "" {
		a.InstallationID = id
	}
}

// BitbucketAccessTokenDetails carries the credential on writes and its
// non-secret metadata on reads (the API never echoes access_token back).
type BitbucketAccessTokenDetails struct {
	AccessToken     string `json:"access_token,omitempty"`
	AccessTokenType string `json:"access_token_type,omitempty"` // PAT | TOKEN
	Username        string `json:"username,omitempty"`
	AccountID       string `json:"account_id,omitempty"`
}

// BitbucketInstallation is a parent Bitbucket connection.
type BitbucketInstallation struct {
	ID                 string                       `json:"id"`
	Name               string                       `json:"name"`
	ServerURL          string                       `json:"server_url,omitempty"`
	ExternalServerURL  string                       `json:"external_server_url,omitempty"`
	AccessTokenDetails *BitbucketAccessTokenDetails `json:"access_token_details,omitempty"`
	IntegrationStatus  string                       `json:"integration_status,omitempty"`
	CloudIntegration   bool                         `json:"cloud_integration"`
}

// BitbucketInstallationWrite is the POST/PATCH body (all fields optional on
// PATCH).
type BitbucketInstallationWrite struct {
	Name               string                       `json:"name,omitempty"`
	ServerURL          string                       `json:"server_url,omitempty"`
	AccessTokenDetails *BitbucketAccessTokenDetails `json:"access_token_details,omitempty"`
}

const bitbucketInstallationsPath = "/api/shiftleft/bitbucket/installations/"

func (client *APIClient) ListBitbucketInstallations() ([]BitbucketInstallation, error) {
	return getAllScmPages[BitbucketInstallation](client, bitbucketInstallationsPath)
}

// GetBitbucketInstallation reads via list-filter (no single-item GET route).
func (client *APIClient) GetBitbucketInstallation(id string) (*BitbucketInstallation, error) {
	all, err := client.ListBitbucketInstallations()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

func (client *APIClient) CreateBitbucketInstallation(body BitbucketInstallationWrite) (*BitbucketInstallation, error) {
	resp, err := client.Post(bitbucketInstallationsPath, body)
	if err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	created := BitbucketInstallation{}
	if err := resp.ReadJSON(&created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (client *APIClient) UpdateBitbucketInstallation(id string, body BitbucketInstallationWrite) (*BitbucketInstallation, error) {
	resp, err := client.Patch(fmt.Sprintf("%s%s/", bitbucketInstallationsPath, id), body)
	if err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	updated := BitbucketInstallation{}
	if err := resp.ReadJSON(&updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (client *APIClient) DeleteBitbucketInstallation(id string) error {
	_, err := client.Delete(fmt.Sprintf("%s%s/", bitbucketInstallationsPath, id))
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}

func bitbucketAccountsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/bitbucket/installations/%s/integrated_accounts/", installationID)
}

// ListBitbucketAccounts fans out across every Bitbucket installation since
// there is no global integrated_accounts list endpoint: list installations,
// then list each installation's integrated_accounts and concatenate.
func (client *APIClient) ListBitbucketAccounts() ([]BitbucketAccount, error) {
	return listScmUnitsByInstallation[BitbucketAccount](client, "/api/shiftleft/bitbucket/installations/", bitbucketAccountsPath)
}

// GetBitbucketAccount reads via list-filter on the installation-scoped list.
func (client *APIClient) GetBitbucketAccount(installationID, accountID string) (*BitbucketAccount, error) {
	return findScmUnit[BitbucketAccount](client, bitbucketAccountsPath(installationID), installationID, accountID)
}

func (client *APIClient) UpdateBitbucketAccount(installationID, accountID string, body ScmInstallationUpdate) (*BitbucketAccount, error) {
	updatePath := fmt.Sprintf("%s%s/", bitbucketAccountsPath(installationID), accountID)
	return updateScmUnit[BitbucketAccount](client, updatePath, bitbucketAccountsPath(installationID), installationID, accountID, body)
}
