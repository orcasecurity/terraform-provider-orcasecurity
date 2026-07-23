package api_client

import "fmt"

type BitbucketAccount struct {
	ID             string `json:"id"`
	InstallationID string `json:"installation_id,omitempty"`
	// AccountID is the Bitbucket-side slug (serializer source:
	// bitbucket_account_slug). Distinct from ID, the Orca unit UUID.
	AccountID   string `json:"account_id,omitempty"`
	AccountName string `json:"account_name"`
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

func (b *BitbucketInstallation) installationID() string { return b.ID }

const bitbucketInstallationsPath = "/api/shiftleft/bitbucket/installations/"

func (client *APIClient) ListBitbucketInstallations() ([]BitbucketInstallation, error) {
	return getAllScmPages[BitbucketInstallation](client, bitbucketInstallationsPath)
}

// GetBitbucketInstallation reads via list-filter. Returns nil when absent.
func (client *APIClient) GetBitbucketInstallation(id string) (*BitbucketInstallation, error) {
	return findScmInstallation[BitbucketInstallation](client, bitbucketInstallationsPath, id)
}

func (client *APIClient) CreateBitbucketInstallation(body BitbucketInstallationWrite) (*BitbucketInstallation, error) {
	return createScmInstallation[BitbucketInstallation](client, bitbucketInstallationsPath, body)
}

// UpdateBitbucketInstallation PATCHes and decodes the response (Bitbucket
// echoes the full serializer on PATCH).
func (client *APIClient) UpdateBitbucketInstallation(id string, body BitbucketInstallationWrite) (*BitbucketInstallation, error) {
	return patchScmInstallation[BitbucketInstallation](client, bitbucketInstallationsPath, id, body)
}

func (client *APIClient) DeleteBitbucketInstallation(id string) error {
	return deleteScmInstallation(client, bitbucketInstallationsPath, id)
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

// GetBitbucketAccount reads via list-filter on the installation-scoped list
// by Orca unit UUID.
func (client *APIClient) GetBitbucketAccount(installationID, orcaAccountID string) (*BitbucketAccount, error) {
	return findScmUnit[BitbucketAccount](client, bitbucketAccountsPath(installationID), installationID, orcaAccountID)
}

// FindBitbucketAccountBySlug reads via list-filter matching Bitbucket account_id (slug).
func (client *APIClient) FindBitbucketAccountBySlug(installationID, slug string) (*BitbucketAccount, error) {
	all, err := getAllScmPages[BitbucketAccount](client, bitbucketAccountsPath(installationID))
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].AccountID == slug {
			all[i].stampInstallationID(installationID)
			return &all[i], nil
		}
	}
	return nil, nil
}

func (client *APIClient) UpdateBitbucketAccount(installationID, orcaAccountID string, body ScmInstallationUpdate) (*BitbucketAccount, error) {
	updatePath := fmt.Sprintf("%s%s/", bitbucketAccountsPath(installationID), orcaAccountID)
	return updateScmUnit[BitbucketAccount](client, updatePath, bitbucketAccountsPath(installationID), installationID, orcaAccountID, body)
}

func (client *APIClient) DeleteBitbucketAccount(installationID, orcaAccountID string) error {
	return deleteScmPathIgnoring404(client, fmt.Sprintf("%s%s/", bitbucketAccountsPath(installationID), orcaAccountID))
}

// BitbucketUnitIntegrate is the scan-all (empty repos) create body for a Bitbucket account.
type BitbucketUnitIntegrate struct {
	InstallationID string
	AccountID      string // Bitbucket slug
	Body           ScmInstallationUpdate
}

func (client *APIClient) IntegrateBitbucketUnit(req BitbucketUnitIntegrate) error {
	body := struct {
		InstallationID        string                  `json:"installation_id"`
		AccountID             string                  `json:"account_id"`
		InstallationMode      string                  `json:"installation_mode,omitempty"`
		DefaultPolicies       bool                    `json:"default_policies"`
		Policies              []string                `json:"policies"`
		ProjectID             string                  `json:"project_id,omitempty"`
		ConfigurationSettings ShiftLeftConfigSettings `json:"configuration_settings"`
		Repositories          []struct{}              `json:"repositories"`
	}{
		InstallationID:        req.InstallationID,
		AccountID:             req.AccountID,
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
	return client.integrateScmRepositories("bitbucket", body)
}
