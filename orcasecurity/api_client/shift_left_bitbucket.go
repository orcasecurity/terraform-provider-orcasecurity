package api_client

import "fmt"

type BitbucketAccount struct {
	ID               string                  `json:"id"`
	InstallationID   string                  `json:"installation_id,omitempty"`
	AccountName      string                  `json:"account_name"`
	InstallationMode string                  `json:"installation_mode,omitempty"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []ScmPolicyRef          `json:"policies,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

// bitbucketInstallation is the minimal shape needed to drive the
// installations -> integrated_accounts fan-out; Bitbucket has no global
// integrated_accounts list endpoint.
type bitbucketInstallation struct {
	ID string `json:"id"`
}

// ListBitbucketAccounts fans out across every Bitbucket installation since
// there is no global integrated_accounts list endpoint: list installations,
// then list each installation's integrated_accounts and concatenate.
func (client *APIClient) ListBitbucketAccounts() ([]BitbucketAccount, error) {
	installations, err := getAllScmPages[bitbucketInstallation](client, "/api/shiftleft/bitbucket/installations/")
	if err != nil {
		return nil, err
	}
	var all []BitbucketAccount
	for _, inst := range installations {
		accs, err := getAllScmPages[BitbucketAccount](client, fmt.Sprintf("/api/shiftleft/bitbucket/installations/%s/integrated_accounts/", inst.ID))
		if err != nil {
			return nil, err
		}
		for i := range accs {
			if accs[i].InstallationID == "" {
				accs[i].InstallationID = inst.ID
			}
		}
		all = append(all, accs...)
	}
	return all, nil
}

// GetBitbucketAccount reads via list-filter on the installation-scoped list.
func (client *APIClient) GetBitbucketAccount(installationID, accountID string) (*BitbucketAccount, error) {
	all, err := getAllScmPages[BitbucketAccount](client, fmt.Sprintf("/api/shiftleft/bitbucket/installations/%s/integrated_accounts/", installationID))
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

func (client *APIClient) UpdateBitbucketAccount(installationID, accountID string, body ScmInstallationUpdate) (*BitbucketAccount, error) {
	if _, err := client.Put(fmt.Sprintf("/api/shiftleft/bitbucket/installations/%s/integrated_accounts/%s/", installationID, accountID), body); err != nil {
		return nil, err
	}
	return client.GetBitbucketAccount(installationID, accountID)
}
