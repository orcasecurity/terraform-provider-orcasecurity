package api_client

import "fmt"

type BitbucketAccount struct {
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

func (a *BitbucketAccount) unitID() string { return a.ID }

func (a *BitbucketAccount) stampInstallationID(id string) {
	if a.InstallationID == "" {
		a.InstallationID = id
	}
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
