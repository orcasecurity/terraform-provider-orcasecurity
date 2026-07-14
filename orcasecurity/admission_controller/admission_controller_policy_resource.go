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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &policyResource{}
	_ resource.ResourceWithConfigure   = &policyResource{}
	_ resource.ResourceWithImportState = &policyResource{}
)

const errCreatingPolicy = "Error creating admission controller policy"
const errReadingPolicy = "Error reading admission controller policy"
const errUpdatingPolicy = "Error updating admission controller policy"

type policyResource struct {
	apiClient *api_client.APIClient
}

type policyResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	IsActive          types.Bool   `tfsdk:"is_active"`
	EnforcementAction types.String `tfsdk:"enforcement_action"`
	Controls          types.List   `tfsdk:"controls"`
}

func NewAdmissionControllerPolicyResource() resource.Resource {
	return &policyResource{}
}

func (r *policyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admission_controller_policy"
}

func (r *policyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *policyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an Admission Controller policy: a named group of controls with an " +
			"enforcement action. Assign policies to clusters with " +
			"`orcasecurity_admission_controller_policy_assignment`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Policy ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Policy name (unique within the organization).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Policy description.",
			},
			"is_active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the policy is active. Defaults to `true`.",
			},
			"enforcement_action": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("monitor"),
				Description: "What happens when a control matches: `monitor` (warn only) or `block` " +
					"(reject the Kubernetes admission request). Defaults to `monitor`.",
				Validators: []validator.String{
					stringvalidator.OneOf("monitor", "block"),
				},
			},
			"controls": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "IDs of `orcasecurity_admission_controller_control` resources attached to this policy.",
			},
		},
	}
}

func policyPayloadFromPlan(ctx context.Context, plan policyResourceModel) api_client.AdmissionControllerPolicy {
	payload := api_client.AdmissionControllerPolicy{
		ID:                plan.ID.ValueString(),
		Name:              plan.Name.ValueString(),
		IsActive:          plan.IsActive.ValueBool(),
		EnforcementAction: plan.EnforcementAction.ValueString(),
		Controls:          stringListToSlice(ctx, plan.Controls),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		payload.Description = &description
	}
	return payload
}

func populatePolicyState(ctx context.Context, state *policyResourceModel, instance *api_client.AdmissionControllerPolicy, resp *resource.ReadResponse) {
	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = stringFromAPI(state.Description, instance.Description)
	state.IsActive = types.BoolValue(instance.IsActive)
	state.EnforcementAction = types.StringValue(instance.EnforcementAction)

	controls, diags := stringListFromAPI(ctx, state.Controls, instance.Controls)
	resp.Diagnostics.Append(diags...)
	state.Controls = controls
}

func (r *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateAdmissionControllerPolicy(policyPayloadFromPlan(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(errCreatingPolicy, "Could not create policy, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.IsActive = types.BoolValue(instance.IsActive)
	plan.EnforcementAction = types.StringValue(instance.EnforcementAction)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetAdmissionControllerPolicy(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(errReadingPolicy,
			fmt.Sprintf("Could not read policy ID %s: %s", state.ID.ValueString(), err.Error()))
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Admission controller policy %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	populatePolicyState(ctx, &state, instance, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.UpdateAdmissionControllerPolicy(policyPayloadFromPlan(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingPolicy, "Could not update policy, unexpected error: "+err.Error())
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.IsActive = types.BoolValue(instance.IsActive)
	plan.EnforcementAction = types.StringValue(instance.EnforcementAction)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteAdmissionControllerPolicy(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting admission controller policy",
			"Could not delete policy, unexpected error: "+err.Error())
	}
}
