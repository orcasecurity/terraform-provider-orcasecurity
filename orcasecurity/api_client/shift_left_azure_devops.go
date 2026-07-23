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

func (client *APIClient) GetAzureDevopsInstallation(id string) (*AzureDevopsInstallation, error) {
	return findScmInstallation[AzureDevopsInstallation](client, azureDevopsInstallationsPath, id)
}

func (client *APIClient) CreateAzureDevopsInstallation(body AzureDevopsInstallationWrite) (*AzureDevopsInstallation, error) {
	return createScmInstallation[AzureDevopsInstallation](client, azureDevopsInstallationsPath, body)
}

// PATCH returns empty body; re-read after update.
func (client *APIClient) UpdateAzureDevopsInstallation(id string, body AzureDevopsInstallationWrite) (*AzureDevopsInstallation, error) {
	return patchScmInstallationAndReread[AzureDevopsInstallation](client, azureDevopsInstallationsPath, id, body)
}

func (client *APIClient) DeleteAzureDevopsInstallation(id string) error {
	return deleteScmInstallation(client, azureDevopsInstallationsPath, id)
}

func azureDevopsAccountsPath(installationID string) string {
	return fmt.Sprintf("/api/shiftleft/azure_devops/installations/%s/integrated_accounts/", installationID)
}

// Global integrated_accounts list omits installation_id; fan out per installation.
func (client *APIClient) ListAzureDevopsAccounts() ([]AzureDevopsAccount, error) {
	return listScmUnitsByInstallation[AzureDevopsAccount](client, "/api/shiftleft/azure_devops/installations/", azureDevopsAccountsPath)
}

func (client *APIClient) GetAzureDevopsAccount(installationID, orcaAccountID string) (*AzureDevopsAccount, error) {
	return findScmUnit[AzureDevopsAccount](client, azureDevopsAccountsPath(installationID), installationID, orcaAccountID)
}

func (client *APIClient) FindAzureDevopsAccountByName(installationID, accountName string) (*AzureDevopsAccount, error) {
	all, err := getAllScmPages[AzureDevopsAccount](client, azureDevopsAccountsPath(installationID))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AccountName == accountName {
			all[i].stampInstallationID(installationID)
			return &all[i], nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateAzureDevopsAccount(installationID, orcaAccountID string, body ScmInstallationUpdate) (*AzureDevopsAccount, error) {
	updatePath := fmt.Sprintf("%s%s/", azureDevopsAccountsPath(installationID), orcaAccountID)
	return updateScmUnit[AzureDevopsAccount](client, updatePath, azureDevopsAccountsPath(installationID), installationID, orcaAccountID, body)
}

func (client *APIClient) DeleteAzureDevopsAccount(installationID, orcaAccountID string) error {
	return deleteScmPathIgnoring404(client, fmt.Sprintf("%s%s/", azureDevopsAccountsPath(installationID), orcaAccountID))
}

type AzureDevopsUnitIntegrate struct {
	InstallationID string
	AccountName    string
	Body           ScmInstallationUpdate
}

func (client *APIClient) IntegrateAzureDevopsUnit(req AzureDevopsUnitIntegrate) error {
	body := struct {
		InstallationID        string                  `json:"installation_id"`
		AzureAccountName      string                  `json:"azure_account_name"`
		InstallationMode      string                  `json:"installation_mode,omitempty"`
		DefaultPolicies       bool                    `json:"default_policies"`
		Policies              []string                `json:"policies"`
		ProjectID             string                  `json:"project_id,omitempty"`
		ConfigurationSettings ShiftLeftConfigSettings `json:"configuration_settings"`
		Repositories          []struct{}              `json:"repositories"`
	}{
		InstallationID:        req.InstallationID,
		AzureAccountName:      req.AccountName,
		InstallationMode:      req.Body.InstallationMode,
		DefaultPolicies:       req.Body.DefaultPolicies,
		Policies:              req.Body.Policies,
		ProjectID:             req.Body.ProjectID,
		ConfigurationSettings: req.Body.ConfigSettings,
		Repositories:          []struct{}{},
	}
	if req.Body.ProjectID != "" {
		body.Policies = nil
	}
	return client.integrateScmRepositories("azure_devops", body)
}
