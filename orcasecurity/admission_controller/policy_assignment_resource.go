package admission_controller

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	Clusters         types.Set    `tfsdk:"clusters"`
	CloudAccounts    types.Set    `tfsdk:"cloud_accounts"`
	PolicyIDs        types.Set    `tfsdk:"policy_ids"`
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

	// An unknown value (e.g. derived from another resource's computed
	// attribute) may become a valid scope at apply time, so validation must
	// be deferred, not failed. resourcevalidator.AtLeastOneOf can't replace
	// this check: full_organization = false must count as absent.
	if config.FullOrganization.IsUnknown() || config.Clusters.IsUnknown() || config.CloudAccounts.IsUnknown() {
		return
	}

	fullOrganization := !config.FullOrganization.IsNull() && config.FullOrganization.ValueBool()
	hasClusters := !config.Clusters.IsNull() && len(config.Clusters.Elements()) > 0
	hasCloudAccounts := !config.CloudAccounts.IsNull() && len(config.CloudAccounts.Elements()) > 0

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
			"clusters": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Kubernetes cluster IDs the policies apply to.",
			},
			"cloud_accounts": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Cloud account IDs whose clusters the policies apply to.",
			},
			"policy_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "IDs of `orcasecurity_admission_controller_policy` resources to assign.",
			},
		},
	}
}

func policyAssignmentPayloadFromPlan(ctx context.Context, plan policyAssignmentResourceModel, diagnostics *diag.Diagnostics) api_client.AdmissionControllerScope {
	clusters, diags := integrations_common.StringSliceFromSet(ctx, plan.Clusters)
	diagnostics.Append(diags...)
	cloudAccounts, diags := integrations_common.StringSliceFromSet(ctx, plan.CloudAccounts)
	diagnostics.Append(diags...)
	policyIDs, diags := integrations_common.StringSliceFromSet(ctx, plan.PolicyIDs)
	diagnostics.Append(diags...)

	payload := api_client.AdmissionControllerScope{
		ID:               plan.ID.ValueString(),
		Name:             plan.Name.ValueString(),
		FullOrganization: plan.FullOrganization.ValueBool(),
		Clusters:         clusters,
		CloudAccounts:    cloudAccounts,
		PolicyIDs:        policyIDs,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		payload.Description = &description
	}
	return payload
}

// populatePolicyAssignmentState maps an API scope onto the model. state is the
// plan (Create/Update) or prior state (Read/import). policy_ids is write-only
// in the API: responses return embedded policies[] objects instead, so the IDs
// are reconstructed from there (keeps imports clean).
func populatePolicyAssignmentState(ctx context.Context, state *policyAssignmentResourceModel, instance *api_client.AdmissionControllerScope, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = integrations_common.OptionalStringMatchPlan(state.Description, instance.Description)
	state.FullOrganization = types.BoolValue(instance.FullOrganization)

	clusters, diags := integrations_common.OptionalSetMatchPlan(ctx, state.Clusters, instance.Clusters)
	diagnostics.Append(diags...)
	state.Clusters = clusters

	cloudAccounts, diags := integrations_common.OptionalSetMatchPlan(ctx, state.CloudAccounts, instance.CloudAccounts)
	diagnostics.Append(diags...)
	state.CloudAccounts = cloudAccounts

	policyIDs := make([]string, 0, len(instance.Policies))
	for _, policy := range instance.Policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	policyIDsSet, diags := integrations_common.OptionalSetMatchPlan(ctx, state.PolicyIDs, policyIDs)
	diagnostics.Append(diags...)
	state.PolicyIDs = policyIDsSet
}

func (r *policyAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := policyAssignmentPayloadFromPlan(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateAdmissionControllerScope(payload)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingAssignment, "Could not create policy assignment, unexpected error: "+err.Error())
		return
	}

	populatePolicyAssignmentState(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
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

	populatePolicyAssignmentState(ctx, &state, instance, &resp.Diagnostics)
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

	payload := policyAssignmentPayloadFromPlan(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.UpdateAdmissionControllerScope(payload)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingAssignment, "Could not update policy assignment, unexpected error: "+err.Error())
		return
	}

	populatePolicyAssignmentState(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
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
