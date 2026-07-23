package api_client

import "fmt"

type AzureDevopsAccount struct {
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

func (a *AzureDevopsAccount) unitID() string { return a.ID }

func (a *AzureDevopsAccount) stampInstallationID(id string) {
	if a.InstallationID == "" {
		a.InstallationID = id
	}
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
