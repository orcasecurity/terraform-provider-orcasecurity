package cloudaccount

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &cloudAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudAccountDataSource{}
)

type cloudAccountStateModel struct {
	CloudAccountID             types.String `tfsdk:"cloud_account_id"`
	Name                       types.String `tfsdk:"name"`
	CloudProviderID            types.String `tfsdk:"cloud_provider_id"`
	CloudVendorID              types.String `tfsdk:"cloud_vendor_id"`
	Description                types.String `tfsdk:"description"`
	CloudAccountStatus         types.String `tfsdk:"cloud_account_status"`
	CloudAccountStatusInfo     types.String `tfsdk:"cloud_account_status_info"`
	CloudProvider              types.String `tfsdk:"cloud_provider"`
	CloudProviderPartition     types.String `tfsdk:"cloud_provider_partition"`
	RoleExternalID             types.String `tfsdk:"role_external_id"`
	AWSRoleArn                 types.String `tfsdk:"aws_role_arn"`
	ScanInAccount              types.Bool   `tfsdk:"scan_inaccount"`
	CreatedTime                types.String `tfsdk:"created_time"`
	VendorID                   types.String `tfsdk:"vendor_id"`
	ScanLimitation             types.String `tfsdk:"scan_limitation"`
	ScanLimitationDetails      types.String `tfsdk:"scan_limitation_details"`
	Type                       types.String `tfsdk:"type"`
	ParentCloudAccountID       types.String `tfsdk:"parent_cloud_account_id"`
	IsMgmt                     types.Bool   `tfsdk:"is_mgmt"`
	ParentCloudAccountVendorID types.String `tfsdk:"parent_cloud_account_vendor_id"`
	ScanMode                   types.String `tfsdk:"scan_mode"`
	AzureTenantID              types.String `tfsdk:"azure_tenant_id"`
	AzureSubscriptionID        types.String `tfsdk:"azure_subscription_id"`
	GCPOrganizationID          types.String `tfsdk:"gcp_organization_id"`
}

type cloudAccountDataSource struct {
	apiClient *api_client.APIClient
}

func NewCloudAccountDataSource() datasource.DataSource {
	return &cloudAccountDataSource{}
}

func (ds *cloudAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *cloudAccountDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

func (ds *cloudAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch cloud account by name or cloud vendor ID. Exactly one of 'name' or 'cloud_vendor_id' must be specified.",
		Attributes: map[string]schema.Attribute{
			"cloud_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud account ID.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Cloud account name to search for. Can be the full name with ID suffix or just the base name. Conflicts with cloud_vendor_id.",
			},
			"cloud_vendor_id": schema.StringAttribute{
				Optional:    true,
				Description: "Cloud vendor ID to search for. Conflicts with name.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud account description.",
			},
			"cloud_account_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the cloud account (e.g., online, offline).",
			},
			"cloud_account_status_info": schema.StringAttribute{
				Computed:    true,
				Description: "Additional information about the cloud account status.",
			},
			"cloud_provider": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud provider type (aws, azure, gcp, etc.).",
			},
			"cloud_provider_id": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud provider specific ID.",
			},
			"cloud_provider_partition": schema.StringAttribute{
				Computed:    true,
				Description: "Cloud provider partition (e.g., public, gov).",
			},
			"role_external_id": schema.StringAttribute{
				Computed:    true,
				Description: "External ID for role assumption.",
			},
			"aws_role_arn": schema.StringAttribute{
				Computed:    true,
				Description: "AWS role ARN for cross-account access.",
			},
			"scan_inaccount": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether scanning is performed in-account.",
			},
			"created_time": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp when the cloud account was created.",
			},
			"vendor_id": schema.StringAttribute{
				Computed:    true,
				Description: "Vendor-specific ID.",
			},
			"scan_limitation": schema.StringAttribute{
				Computed:    true,
				Description: "Scan limitations for this account.",
			},
			"scan_limitation_details": schema.StringAttribute{
				Computed:    true,
				Description: "Detailed information about scan limitations.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Type of cloud account (e.g., REGULAR).",
			},
			"parent_cloud_account_id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the parent cloud account.",
			},
			"is_mgmt": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a management account.",
			},
			"parent_cloud_account_vendor_id": schema.StringAttribute{
				Computed:    true,
				Description: "Vendor ID of the parent cloud account.",
			},
			"scan_mode": schema.StringAttribute{
				Computed:    true,
				Description: "Scan mode for this account.",
			},
			"azure_tenant_id": schema.StringAttribute{
				Computed:    true,
				Description: "Azure tenant ID (for Azure accounts).",
			},
			"azure_subscription_id": schema.StringAttribute{
				Computed:    true,
				Description: "Azure subscription ID (for Azure accounts).",
			},
			"gcp_organization_id": schema.StringAttribute{
				Computed:    true,
				Description: "GCP organization ID (for GCP accounts).",
			},
		},
	}
}

func (ds *cloudAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state cloudAccountStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one search parameter is provided
	hasName := !state.Name.IsNull() && !state.Name.IsUnknown() && state.Name.ValueString() != ""
	hasVendorID := !state.CloudVendorID.IsNull() && !state.CloudVendorID.IsUnknown() && state.CloudVendorID.ValueString() != ""

	if !hasName && !hasVendorID {
		resp.Diagnostics.AddError(
			"Missing required parameter",
			"Either 'name' or 'cloud_vendor_id' must be specified.",
		)
		return
	}

	if hasName && hasVendorID {
		resp.Diagnostics.AddError(
			"Conflicting parameters",
			"Only one of 'name' or 'cloud_vendor_id' can be specified, not both.",
		)
		return
	}

	var item *api_client.CloudAccount
	var err error

	// Search by the provided parameter
	if hasName {
		item, err = ds.apiClient.GetCloudAccountByName(state.Name.ValueString())
	} else {
		item, err = ds.apiClient.GetCloudAccountByVendorID(state.CloudVendorID.ValueString())
	}

	if err != nil {
		resp.Diagnostics.AddError("Unable to read cloud account", err.Error())
		return
	}

	// Map all fields
	state.CloudAccountID = types.StringValue(item.CloudAccountID)
	state.Name = types.StringValue(item.Name)
	state.CloudProviderID = types.StringValue(item.CloudProviderID)
	state.CloudVendorID = types.StringValue(item.CloudVendorID)
	state.Description = types.StringValue(item.Description)
	state.CloudAccountStatus = types.StringValue(item.CloudAccountStatus)
	state.CloudAccountStatusInfo = types.StringValue(item.CloudAccountStatusInfo)
	state.CloudProvider = types.StringValue(item.CloudProvider)
	state.CloudProviderPartition = types.StringValue(item.CloudProviderPartition)
	state.RoleExternalID = types.StringValue(item.RoleExternalID)
	state.AWSRoleArn = types.StringValue(item.AWSRoleArn)
	state.ScanInAccount = types.BoolValue(item.ScanInAccount)
	state.CreatedTime = types.StringValue(item.CreatedTime)
	state.VendorID = types.StringValue(item.VendorID)
	state.ScanLimitation = types.StringValue(item.ScanLimitation)
	state.ScanLimitationDetails = types.StringValue(item.ScanLimitationDetails)
	state.Type = types.StringValue(item.Type)
	state.ParentCloudAccountID = types.StringValue(item.ParentCloudAccountID)
	state.IsMgmt = types.BoolValue(item.IsMgmt)
	state.ParentCloudAccountVendorID = types.StringValue(item.ParentCloudAccountVendorID)
	state.ScanMode = types.StringValue(item.ScanMode)
	state.AzureTenantID = types.StringValue(item.AzureTenantID)
	state.AzureSubscriptionID = types.StringValue(item.AzureSubscriptionID)

	// Handle nullable GCP organization ID
	if item.GCPOrganizationID != nil {
		state.GCPOrganizationID = types.StringValue(*item.GCPOrganizationID)
	} else {
		state.GCPOrganizationID = types.StringNull()
	}

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
