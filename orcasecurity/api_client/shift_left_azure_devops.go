package api_client

import "fmt"

type AzureDevopsAccount struct {
	ID             string `json:"id"`
	InstallationID string `json:"installation_id,omitempty"`
	AccountName    string `json:"account_name"`
	ScmUnitCommonFields
}

func (a *AzureDevopsAccount) unitID() string { return a.ID }

func (a *AzureDevopsAccount) stampInstallationID(id string) {
	if a.InstallationID == "" {
		a.InstallationID = id
	}
}

// AzureAccessTokenDetails carries the credential on writes; the API never
// echoes access_token back (reads expose access_token_account_name instead).
type AzureAccessTokenDetails struct {
	AccessToken string `json:"access_token"`
	AccountName string `json:"account_name,omitempty"`
}

// AzureDevopsInstallation is a parent Azure DevOps connection.
type AzureDevopsInstallation struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	ServerURL              string `json:"server_url,omitempty"`
	ExternalServerURL      string `json:"external_server_url,omitempty"`
	AccessTokenType        string `json:"access_token_type,omitempty"` // SINGLE_ACCOUNT | ALL_ACCOUNTS
	AccessTokenAccountName string `json:"access_token_account_name,omitempty"`
	IntegrationStatus      string `json:"integration_status,omitempty"`
	CloudIntegration       bool   `json:"cloud_integration"`
}

// AzureDevopsInstallationWrite is the POST/PATCH body.
type AzureDevopsInstallationWrite struct {
	Name               string                   `json:"name,omitempty"`
	ServerURL          string                   `json:"server_url,omitempty"`
	AccessTokenDetails *AzureAccessTokenDetails `json:"access_token_details,omitempty"`
}

func (a *AzureDevopsInstallation) installationID() string { return a.ID }

const azureDevopsInstallationsPath = "/api/shiftleft/azure_devops/installations/"

func (client *APIClient) ListAzureDevopsInstallations() ([]AzureDevopsInstallation, error) {
	return getAllScmPages[AzureDevopsInstallation](client, azureDevopsInstallationsPath)
}

// GetAzureDevopsInstallation reads via list-filter. Returns nil when absent.
func (client *APIClient) GetAzureDevopsInstallation(id string) (*AzureDevopsInstallation, error) {
	return findScmInstallation[AzureDevopsInstallation](client, azureDevopsInstallationsPath, id)
}

func (client *APIClient) CreateAzureDevopsInstallation(body AzureDevopsInstallationWrite) (*AzureDevopsInstallation, error) {
	return createScmInstallation[AzureDevopsInstallation](client, azureDevopsInstallationsPath, body)
}

// UpdateAzureDevopsInstallation PATCHes and re-reads (the PATCH response body
// is empty).
func (client *APIClient) UpdateAzureDevopsInstallation(id string, body AzureDevopsInstallationWrite) (*AzureDevopsInstallation, error) {
	return patchScmInstallationAndReread[AzureDevopsInstallation](client, azureDevopsInstallationsPath, id, body)
}

func (client *APIClient) DeleteAzureDevopsInstallation(id string) error {
	return deleteScmInstallation(client, azureDevopsInstallationsPath, id)
}

func azureDevopsAccountsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/azure_devops/installations/%s/integrated_accounts/", installationID)
}

// ListAzureDevopsAccounts fans out across every Azure DevOps installation so
// each account carries its installation_id (the global
// /azure_devops/integrated_accounts/ endpoint omits it, which breaks the
// config-resource for_each workflow).
func (client *APIClient) ListAzureDevopsAccounts() ([]AzureDevopsAccount, error) {
	return listScmUnitsByInstallation[AzureDevopsAccount](client, "/api/shiftleft/azure_devops/installations/", azureDevopsAccountsPath)
}

// GetAzureDevopsAccount reads via list-filter on the installation-scoped list.
func (client *APIClient) GetAzureDevopsAccount(installationID, accountID string) (*AzureDevopsAccount, error) {
	return findScmUnit[AzureDevopsAccount](client, azureDevopsAccountsPath(installationID), installationID, accountID)
}

func (client *APIClient) UpdateAzureDevopsAccount(installationID, accountID string, body ScmInstallationUpdate) (*AzureDevopsAccount, error) {
	updatePath := fmt.Sprintf("%s%s/", azureDevopsAccountsPath(installationID), accountID)
	return updateScmUnit[AzureDevopsAccount](client, updatePath, azureDevopsAccountsPath(installationID), installationID, accountID, body)
}
