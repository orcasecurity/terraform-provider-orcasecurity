package api_client

import (
	"fmt"
	"net/url"
	"strings"
)

type CloudAccount struct {
	CloudAccountID             string                 `json:"cloud_account_id"`
	Name                       string                 `json:"name"`
	Description                string                 `json:"description"`
	CloudAccountStatus         string                 `json:"cloud_account_status"`
	CloudAccountStatusInfo     string                 `json:"cloud_account_status_info"`
	CloudProvider              string                 `json:"cloud_provider"`
	CloudProviderID            string                 `json:"cloud_provider_id"`
	CloudProviderPartition     string                 `json:"cloud_provider_partition"`
	RoleExternalID             string                 `json:"role_external_id"`
	AWSRoleArn                 string                 `json:"aws_role_arn"`
	ScanInAccount              bool                   `json:"scan_inaccount"`
	CreatedTime                string                 `json:"created_time"`
	VendorID                   string                 `json:"vendor_id"`
	CloudVendorID              string                 `json:"cloud_vendor_id"`
	ScanLimitation             string                 `json:"scan_limitation"`
	ScanLimitationDetails      string                 `json:"scan_limitation_details"`
	Type                       string                 `json:"type"`
	ParentCloudAccountID       string                 `json:"parent_cloud_account_id"`
	IsMgmt                     bool                   `json:"is_mgmt"`
	ParentCloudAccountVendorID string                 `json:"parent_cloud_account_vendor_id"`
	AccountTags                map[string]interface{} `json:"account_tags"`
	ActivityLogInfo            ActivityLogInfo        `json:"activity_log_info"`
	AuditLogInfo               map[string]interface{} `json:"audit_log_info"`
	DSPMSetupInfo              DSPMSetupInfo          `json:"dspm_setup_info"`
	RemediationConfiguration   interface{}            `json:"remediation_configuration"`
	ScanMode                   string                 `json:"scan_mode"`
	AzureTenantID              string                 `json:"azure_tenant_id"`
	AzureSubscriptionID        string                 `json:"azure_subscription_id"`
	GCPOrganizationID          *string                `json:"gcp_organization_id"`
	K8sClusters                []interface{}          `json:"k8s_clusters"`
	TrailsInfo                 []interface{}          `json:"trails_info"`
}

type ActivityLogInfo struct {
	Onboarded bool     `json:"onboarded"`
	Reasons   []string `json:"reasons"`
}

type DSPMSetupInfo struct {
	FeatureStatus          string  `json:"feature_status"`
	Deployed               bool    `json:"deployed"`
	AzureTenantID          string  `json:"azure_tenant_id"`
	AzureSubscriptionID    string  `json:"azure_subscription_id"`
	AzureResourceGroupName *string `json:"azure_resource_group_name"`
	AzureMgntGroupName     string  `json:"azure_mgnt_group_name"`
	URL                    string  `json:"url"`
	FailureReason          string  `json:"failure_reason"`
}

type CloudAccountResponse struct {
	Version string         `json:"version"`
	Status  string         `json:"status"`
	Data    []CloudAccount `json:"data"`
}

type CloudAccountQueryParams struct {
	GetStats                    *bool
	GetTotalStats               *bool
	SortBy                      string
	SortOrder                   string
	Filter                      string
	UniqueList                  string
	GetTags                     *bool
	WithAssociatedBusinessUnits *bool
	IncludeAllAccountTypes      *bool
	FreeTextSearch              string
	CloudProvider               string
}

// extractNameFromConcatenated extracts the base name by removing the specific ID suffix
func extractNameFromConcatenated(fullName, accountID string) string {
	suffix := fmt.Sprintf(" (%s)", accountID)
	if strings.HasSuffix(fullName, suffix) {
		return strings.TrimSpace(strings.TrimSuffix(fullName, suffix))
	}
	return fullName // Return original if no suffix match
}

// buildCloudAccountQuery builds the query string from parameters
func buildCloudAccountQuery(params CloudAccountQueryParams) string {
	query := url.Values{}

	if params.GetStats != nil {
		query.Set("get_stats", fmt.Sprintf("%t", *params.GetStats))
	}
	if params.GetTotalStats != nil {
		query.Set("get_total_stats", fmt.Sprintf("%t", *params.GetTotalStats))
	}
	if params.SortBy != "" {
		query.Set("sort_by", params.SortBy)
	}
	if params.SortOrder != "" {
		query.Set("sort_order", params.SortOrder)
	}
	if params.Filter != "" {
		query.Set("filter", params.Filter)
	}
	if params.UniqueList != "" {
		query.Set("unique_list", params.UniqueList)
	}
	if params.GetTags != nil {
		query.Set("get_tags", fmt.Sprintf("%t", *params.GetTags))
	}
	if params.WithAssociatedBusinessUnits != nil {
		query.Set("with_associated_business_units", fmt.Sprintf("%t", *params.WithAssociatedBusinessUnits))
	}
	if params.IncludeAllAccountTypes != nil {
		query.Set("include_all_account_types", fmt.Sprintf("%t", *params.IncludeAllAccountTypes))
	}
	if params.FreeTextSearch != "" {
		query.Set("free_text_search", params.FreeTextSearch)
	}
	if params.CloudProvider != "" {
		query.Set("cloud_provider", params.CloudProvider)
	}

	return query.Encode()
}

// GetCloudAccountByName searches for a cloud account by name (handles concatenated names)
func (client *APIClient) GetCloudAccountByName(name string) (*CloudAccount, error) {
	params := CloudAccountQueryParams{
		GetTags:                     boolPtr(false),
		WithAssociatedBusinessUnits: boolPtr(false),
		IncludeAllAccountTypes:      boolPtr(false),
		FreeTextSearch:              name,
	}

	queryString := buildCloudAccountQuery(params)
	resp, err := client.Get(fmt.Sprintf("/api/cloudaccount?%s", queryString))
	if err != nil {
		return nil, err
	}

	response := CloudAccountResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("cloud account with name '%s' does not exist", name)
	}

	// Find exact match - try both full name and extracted name
	for _, account := range response.Data {
		// First try exact match with full name
		if account.Name == name {
			return &account, nil
		}

		// Then try matching against extracted name (using the cloud_vendor_id, not cloud_account_id)
		extractedName := extractNameFromConcatenated(account.Name, account.CloudVendorID)
		if extractedName == name {
			return &account, nil
		}
	}

	return nil, fmt.Errorf("cloud account with name '%s' does not exist", name)
}

// GetCloudAccountByVendorID searches for a cloud account by cloud vendor ID
func (client *APIClient) GetCloudAccountByVendorID(vendorID string) (*CloudAccount, error) {
	params := CloudAccountQueryParams{
		GetTags:                     boolPtr(false),
		WithAssociatedBusinessUnits: boolPtr(false),
		IncludeAllAccountTypes:      boolPtr(false),
		FreeTextSearch:              vendorID,
	}

	queryString := buildCloudAccountQuery(params)
	resp, err := client.Get(fmt.Sprintf("/api/cloudaccount?%s", queryString))
	if err != nil {
		return nil, err
	}

	response := CloudAccountResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("cloud account with vendor ID '%s' does not exist", vendorID)
	}

	// Find exact match by cloud vendor ID
	for _, account := range response.Data {
		if account.CloudVendorID == vendorID {
			return &account, nil
		}
	}

	return nil, fmt.Errorf("cloud account with vendor ID '%s' does not exist", vendorID)
}

// GetCloudAccountByID retrieves a cloud account by its ID
func (client *APIClient) GetCloudAccountByID(id string) (*CloudAccount, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/cloudaccount/%s", url.PathEscape(id)),
	)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 404 {
		return nil, fmt.Errorf("cloud account with ID '%s' does not exist", id)
	}

	var account CloudAccount
	err = resp.ReadJSON(&account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// ListCloudAccounts retrieves cloud accounts with flexible filtering
func (client *APIClient) ListCloudAccounts(params CloudAccountQueryParams) ([]CloudAccount, error) {
	queryString := buildCloudAccountQuery(params)
	resp, err := client.Get(fmt.Sprintf("/api/cloudaccount?%s", queryString))
	if err != nil {
		return nil, err
	}

	response := CloudAccountResponse{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

// ListCloudAccountsSimple is a convenience method with common defaults
func (client *APIClient) ListCloudAccountsSimple(search string, cloudProvider string) ([]CloudAccount, error) {
	params := CloudAccountQueryParams{
		GetTags:                     boolPtr(false),
		WithAssociatedBusinessUnits: boolPtr(false),
		IncludeAllAccountTypes:      boolPtr(false),
		FreeTextSearch:              search,
		CloudProvider:               cloudProvider,
	}

	return client.ListCloudAccounts(params)
}

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}
