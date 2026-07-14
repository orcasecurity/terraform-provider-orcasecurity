package admission_controller

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &policyAssignmentResource{}
	_ resource.ResourceWithConfigure      = &policyAssignmentResource{}
	_ resource.ResourceWithImportState    = &policyAssignmentResource{}
	_ resource.ResourceWithValidateConfig = &policyAssignmentResource{}
)

const errCreatingAssignment = "Error creating admission controller policy assignment"
const errReadingAssignment = "Error reading admission controller policy assignment"
const errUpdatingAssignment = "Error updating admission controller policy assignment"

type policyAssignmentResource struct {
	apiClient *api_client.APIClient
}

type policyAssignmentResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	FullOrganization types.Bool   `tfsdk:"full_organization"`
	Clusters         types.List   `tfsdk:"clusters"`
	CloudAccounts    types.List   `tfsdk:"cloud_accounts"`
	PolicyIDs        types.List   `tfsdk:"policy_ids"`
}

func NewAdmissionControllerPolicyAssignmentResource() resource.Resource {
	return &policyAssignmentResource{}
}

func (r *policyAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admission_controller_policy_assignment"
}

func (r *policyAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *policyAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ValidateConfig requires a scope target: full_organization, clusters, or
// cloud_accounts. The API accepts an empty scope but it would never match
// anything, and the Orca UI enforces the same rule.
func (r *policyAssignmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config policyAssignmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullOrganization := !config.FullOrganization.IsNull() && !config.FullOrganization.IsUnknown() && config.FullOrganization.ValueBool()
	hasClusters := !config.Clusters.IsNull() && !config.Clusters.IsUnknown() && len(config.Clusters.Elements()) > 0
	hasCloudAccounts := !config.CloudAccounts.IsNull() && !config.CloudAccounts.IsUnknown() && len(config.CloudAccounts.Elements()) > 0

	if !fullOrganization && !hasClusters && !hasCloudAccounts {
		resp.Diagnostics.AddError(
			"Missing policy assignment scope",
			"At least one of `full_organization = true`, `clusters`, or `cloud_accounts` must be set.",
		)
	}
}

func (r *policyAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an Admission Controller policy assignment (the API calls it a \"scope\"): " +
			"attaches policies to Kubernetes clusters, cloud accounts, or the whole organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Policy assignment ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Policy assignment name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Policy assignment description.",
			},
			"full_organization": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Apply the attached policies to every cluster in the organization. Defaults to `false`.",
			},
			"clusters": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes cluster IDs the policies apply to.",
			},
			"cloud_accounts": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Cloud account IDs whose clusters the policies apply to.",
			},
			"policy_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "IDs of `orcasecurity_admission_controller_policy` resources to assign.",
			},
		},
	}
}

func policyAssignmentPayloadFromPlan(ctx context.Context, plan policyAssignmentResourceModel) api_client.AdmissionControllerScope {
	payload := api_client.AdmissionControllerScope{
		ID:               plan.ID.ValueString(),
		Name:             plan.Name.ValueString(),
		FullOrganization: plan.FullOrganization.ValueBool(),
		Clusters:         stringListToSlice(ctx, plan.Clusters),
		CloudAccounts:    stringListToSlice(ctx, plan.CloudAccounts),
		PolicyIDs:        stringListToSlice(ctx, plan.PolicyIDs),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		payload.Description = &description
	}
	return payload
}

// populatePolicyAssignmentState maps an API scope onto the model. policy_ids
// is write-only in the API: reads return embedded policies[] objects instead,
// so the IDs are reconstructed from there (keeps imports clean).
func populatePolicyAssignmentState(ctx context.Context, state *policyAssignmentResourceModel, instance *api_client.AdmissionControllerScope, resp *resource.ReadResponse) {
	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = stringFromAPI(state.Description, instance.Description)
	state.FullOrganization = types.BoolValue(instance.FullOrganization)

	clusters, diags := stringListFromAPI(ctx, state.Clusters, instance.Clusters)
	resp.Diagnostics.Append(diags...)
	state.Clusters = clusters

	cloudAccounts, diags := stringListFromAPI(ctx, state.CloudAccounts, instance.CloudAccounts)
	resp.Diagnostics.Append(diags...)
	state.CloudAccounts = cloudAccounts

	policyIDs := make([]string, 0, len(instance.Policies))
	for _, policy := range instance.Policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	policyIDsList, diags := stringListFromAPI(ctx, state.PolicyIDs, policyIDs)
	resp.Diagnostics.Append(diags...)
	state.PolicyIDs = policyIDsList
}

func (r *policyAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateAdmissionControllerScope(policyAssignmentPayloadFromPlan(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(errCreatingAssignment, "Could not create policy assignment, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.FullOrganization = types.BoolValue(instance.FullOrganization)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetAdmissionControllerScope(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(errReadingAssignment,
			fmt.Sprintf("Could not read policy assignment ID %s: %s", state.ID.ValueString(), err.Error()))
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Admission controller policy assignment %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	populatePolicyAssignmentState(ctx, &state, instance, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *policyAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.UpdateAdmissionControllerScope(policyAssignmentPayloadFromPlan(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingAssignment, "Could not update policy assignment, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.FullOrganization = types.BoolValue(instance.FullOrganization)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteAdmissionControllerScope(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting admission controller policy assignment",
			"Could not delete policy assignment, unexpected error: "+err.Error())
	}
}
