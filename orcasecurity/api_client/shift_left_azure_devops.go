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

// ListAzureDevopsAccounts fans out across every Azure DevOps installation so
// each account carries its installation_id (the global
// /azure_devops/integrated_accounts/ endpoint omits it, which breaks the
// config-resource for_each workflow).
func (client *APIClient) ListAzureDevopsAccounts() ([]AzureDevopsAccount, error) {
	return listScmUnitsByInstallation[AzureDevopsAccount](
		client,
		"/api/shiftleft/azure_devops/installations/",
		func(installationID string) string {
			return fmt.Sprintf("/api/shiftleft/azure_devops/installations/%s/integrated_accounts/", installationID)
		},
		func(a *AzureDevopsAccount, installationID string) {
			if a.InstallationID == "" {
				a.InstallationID = installationID
			}
		},
	)
}

// GetAzureDevopsAccount reads via list-filter on the installation-scoped list.
func (client *APIClient) GetAzureDevopsAccount(installationID, accountID string) (*AzureDevopsAccount, error) {
	all, err := getAllScmPages[AzureDevopsAccount](client, fmt.Sprintf("/api/shiftleft/azure_devops/installations/%s/integrated_accounts/", installationID))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].ID == accountID {
			if all[i].InstallationID == "" {
				all[i].InstallationID = installationID
			}
			return &all[i], nil
		}
	}
	return nil, nil // not found -> caller treats nil as drift
}

func (client *APIClient) UpdateAzureDevopsAccount(installationID, accountID string, body ScmInstallationUpdate) (*AzureDevopsAccount, error) {
	if _, err := client.Put(fmt.Sprintf("/api/shiftleft/azure_devops/installations/%s/integrated_accounts/%s/", installationID, accountID), body); err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	return client.GetAzureDevopsAccount(installationID, accountID)
}
