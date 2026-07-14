package admission_controller

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &controlResource{}
	_ resource.ResourceWithConfigure   = &controlResource{}
	_ resource.ResourceWithImportState = &controlResource{}
)

const errCreatingControl = "Error creating admission controller control"
const errReadingControl = "Error reading admission controller control"
const errUpdatingControl = "Error updating admission controller control"

type controlResource struct {
	apiClient *api_client.APIClient
}

type controlClusterScopeKindModel struct {
	Kinds     types.List `tfsdk:"kinds"`
	APIGroups types.List `tfsdk:"api_groups"`
	Versions  types.List `tfsdk:"versions"`
}

type controlClusterScopeModel struct {
	Kinds []controlClusterScopeKindModel `tfsdk:"kinds"`
}

type controlResourceModel struct {
	ID              types.String              `tfsdk:"id"`
	Name            types.String              `tfsdk:"name"`
	Description     types.String              `tfsdk:"description"`
	TemplateID      types.String              `tfsdk:"template_id"`
	TemplateName    types.String              `tfsdk:"template_name"`
	ClusterScope    *controlClusterScopeModel `tfsdk:"cluster_scope"`
	InputParameters jsontypes.Normalized      `tfsdk:"input_parameters"`
}

func NewAdmissionControllerControlResource() resource.Resource {
	return &controlResource{}
}

func (r *controlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admission_controller_control"
}

func (r *controlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *controlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *controlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an Admission Controller control: one template instantiated with concrete " +
			"parameters and a Kubernetes resource scope. Attach controls to an " +
			"`orcasecurity_admission_controller_policy` to activate them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Control ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Control name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Control description.",
			},
			"template_id": schema.StringAttribute{
				Required: true,
				Description: "ID of the control template to instantiate. Use the " +
					"`orcasecurity_admission_controller_template` data source to look it up by name.",
			},
			"template_name": schema.StringAttribute{
				Computed:    true,
				Description: "Internal name of the template backing this control.",
			},
			"cluster_scope": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Kubernetes resources the control applies to.",
				Attributes: map[string]schema.Attribute{
					"kinds": schema.ListNestedAttribute{
						Required:    true,
						Description: "List of kind selectors.",
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"kinds": schema.ListAttribute{
									Required:    true,
									ElementType: types.StringType,
									Description: "Kubernetes kinds (e.g. `Pod`, `Deployment`). Must be within the template's `supported_kinds`.",
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
								},
								"api_groups": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Kubernetes API groups. Use `[\"\"]` for the core group (recommended; matches the Orca UI).",
								},
								"versions": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Description: "Kubernetes API versions. Use `[\"\"]` for any version (recommended; matches the Orca UI).",
								},
							},
						},
					},
				},
			},
			"input_parameters": schema.StringAttribute{
				Optional:   true,
				CustomType: jsontypes.NormalizedType{},
				Description: "Template-specific parameters as a JSON object (use `jsonencode(...)`). " +
					"The expected fields are defined by the template's schema " +
					"(GET /api/admission_controller/templates/{id}, `content.spec.crd.spec.validation.openAPIV3Schema`). " +
					"Omit for templates without parameters.",
			},
		},
	}
}

// controlPayloadFromPlan builds the API payload from the plan.
func controlPayloadFromPlan(ctx context.Context, plan controlResourceModel) (api_client.AdmissionControllerControl, error) {
	payload := api_client.AdmissionControllerControl{
		ID:         plan.ID.ValueString(),
		Name:       plan.Name.ValueString(),
		TemplateID: plan.TemplateID.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		description := plan.Description.ValueString()
		payload.Description = &description
	}
	if plan.ClusterScope != nil {
		for _, kind := range plan.ClusterScope.Kinds {
			payload.ClusterScope.Kinds = append(payload.ClusterScope.Kinds, api_client.AdmissionControllerClusterScopeKind{
				Kinds:     stringListToSlice(ctx, kind.Kinds),
				APIGroups: stringListToSlice(ctx, kind.APIGroups),
				Versions:  stringListToSlice(ctx, kind.Versions),
			})
		}
	}

	raw, diags := integrations_common.DecodeJSONField(plan.InputParameters, "input_parameters")
	if diags.HasError() {
		return payload, fmt.Errorf("input_parameters must be a JSON object")
	}
	if raw == nil {
		raw = []byte(`{}`)
	}
	payload.InputParameters = raw
	return payload, nil
}

// populateControlState maps an API instance onto the model. prior is the
// pre-refresh state (zero-value model on import).
func populateControlState(ctx context.Context, state *controlResourceModel, instance *api_client.AdmissionControllerControl, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = stringFromAPI(state.Description, instance.Description)
	state.TemplateID = types.StringValue(instance.TemplateID)
	state.TemplateName = types.StringValue(instance.TemplateName)

	scope := &controlClusterScopeModel{}
	for i, kind := range instance.ClusterScope.Kinds {
		var priorKind controlClusterScopeKindModel
		if state.ClusterScope != nil && i < len(state.ClusterScope.Kinds) {
			priorKind = state.ClusterScope.Kinds[i]
		}
		kinds, diags := types.ListValueFrom(ctx, types.StringType, kind.Kinds)
		diagnostics.Append(diags...)
		apiGroups, diags := stringListFromAPI(ctx, priorKind.APIGroups, kind.APIGroups)
		diagnostics.Append(diags...)
		versions, diags := stringListFromAPI(ctx, priorKind.Versions, kind.Versions)
		diagnostics.Append(diags...)
		scope.Kinds = append(scope.Kinds, controlClusterScopeKindModel{
			Kinds:     kinds,
			APIGroups: apiGroups,
			Versions:  versions,
		})
	}
	state.ClusterScope = scope

	inputParameters, diags := integrations_common.EncodeJSONField(instance.InputParameters, state.InputParameters)
	diagnostics.Append(diags...)
	state.InputParameters = inputParameters
}

func (r *controlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan controlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := controlPayloadFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingControl, err.Error())
		return
	}

	instance, err := r.apiClient.CreateAdmissionControllerControl(payload)
	if err != nil {
		resp.Diagnostics.AddError(errCreatingControl, "Could not create control, unexpected error: "+err.Error())
		return
	}

	populateControlState(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *controlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state controlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetAdmissionControllerControl(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(errReadingControl,
			fmt.Sprintf("Could not read control ID %s: %s", state.ID.ValueString(), err.Error()))
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Admission controller control %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	populateControlState(ctx, &state, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *controlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan controlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := controlPayloadFromPlan(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingControl, err.Error())
		return
	}

	instance, err := r.apiClient.UpdateAdmissionControllerControl(payload)
	if err != nil {
		resp.Diagnostics.AddError(errUpdatingControl, "Could not update control, unexpected error: "+err.Error())
		return
	}

	populateControlState(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *controlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state controlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteAdmissionControllerControl(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting admission controller control",
			"Could not delete control, unexpected error: "+err.Error())
	}
}
