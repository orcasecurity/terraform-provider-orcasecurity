package api_client

import "fmt"

type AzureDevopsAccount struct {
	ID               string                  `json:"id"`
	InstallationID   string                  `json:"installation_id,omitempty"`
	AccountName      string                  `json:"account_name"`
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []ScmPolicyRef          `json:"policies,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

func (client *APIClient) ListAzureDevopsAccounts() ([]AzureDevopsAccount, error) {
	return getAllScmPages[AzureDevopsAccount](client, "/api/shiftleft/azure_devops/integrated_accounts/")
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
	return client.GetAzureDevopsAccount(installationID, accountID)
}
