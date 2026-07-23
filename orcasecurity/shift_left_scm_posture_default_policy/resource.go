package shift_left_scm_posture_default_policy

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &defaultPolicyResource{}
	_ resource.ResourceWithConfigure   = &defaultPolicyResource{}
	_ resource.ResourceWithImportState = &defaultPolicyResource{}
)

const (
	errReadDefaultPolicy   = "Error reading SCM posture default policy"
	errUpdateDefaultPolicy = "Error updating SCM posture default policy"
)

type defaultPolicyResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &defaultPolicyResource{} }

type controlModel struct {
	ID       types.String `tfsdk:"id"`
	Disabled types.Bool   `tfsdk:"disabled"`
	Priority types.String `tfsdk:"priority"`
}

type resourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Disabled    types.Bool     `tfsdk:"disabled"`
	Controls    []controlModel `tfsdk:"controls"`
}

func (r *defaultPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_scm_posture_default_policy"
}

func (r *defaultPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *defaultPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages the org-wide built-in SCM Posture policy singleton (GET/PUT /api/shiftleft/scm_posture/policy/). " +
			"The policy always exists in every Orca org, so this resource adopts it: Create and Update both PUT the configuration, " +
			"and Delete only forgets the resource from state (the built-in policy can never be deleted). " +
			"Its name and description are locked server-side; only `disabled` and control overrides are writable.",
		Attributes: map[string]rschema.Attribute{
			"id": rschema.StringAttribute{
				Computed:      true,
				Description:   "Policy UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": rschema.StringAttribute{
				Computed:    true,
				Description: "Policy name (read-only, server-managed).",
			},
			"description": rschema.StringAttribute{
				Computed:    true,
				Description: "Policy description (read-only, server-managed).",
			},
			"disabled": rschema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the policy is disabled. Omit to leave the live value unchanged.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"controls": rschema.ListNestedAttribute{
				Optional: true,
				Description: "Control overrides for SCM posture catalog controls. Omit to leave the live overrides unchanged; " +
					"set to `[]` to clear all overrides.",
				NestedObject: rschema.NestedAttributeObject{
					Attributes: map[string]rschema.Attribute{
						"id": rschema.StringAttribute{
							Required:    true,
							Description: "Catalog control ID.",
						},
						"disabled": rschema.BoolAttribute{
							Optional:    true,
							Description: "Disable this control.",
						},
						"priority": rschema.StringAttribute{
							Optional:    true,
							Description: "Override the control priority.",
							Validators: []validator.String{
								stringvalidator.OneOf("INFO", "LOW", "MEDIUM", "HIGH", "CRITICAL"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *defaultPolicyResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	live, err := r.apiClient.GetScmPostureDefaultPolicy()
	if err != nil {
		resp.Diagnostics.AddError(errReadDefaultPolicy, err.Error())
		return
	}
	state, err := apiToState(live)
	if err != nil {
		resp.Diagnostics.AddError(errReadDefaultPolicy, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *defaultPolicyResource) write(plan *resourceModel, diags *diag.Diagnostics) *resourceModel {
	live, err := r.apiClient.GetScmPostureDefaultPolicy()
	if err != nil {
		diags.AddError(errReadDefaultPolicy, err.Error())
		return nil
	}

	body := api_client.ScmPostureDefaultPolicyWrite{Disabled: live.Disabled}
	if !plan.Disabled.IsNull() && !plan.Disabled.IsUnknown() {
		body.Disabled = plan.Disabled.ValueBool()
	}
	if plan.Controls != nil {
		body.PolicyData.Controls = controlsToAPI(plan.Controls)
	} else {
		existing, err := liveControls(live)
		if err != nil {
			diags.AddError(errUpdateDefaultPolicy, err.Error())
			return nil
		}
		body.PolicyData.Controls = existing
	}

	updated, err := r.apiClient.UpdateScmPostureDefaultPolicy(body)
	if err != nil {
		diags.AddError(errUpdateDefaultPolicy, err.Error())
		return nil
	}
	state, err := apiToState(updated)
	if err != nil {
		diags.AddError(errUpdateDefaultPolicy, err.Error())
		return nil
	}
	if plan.Controls != nil {
		state.Controls = plan.Controls
	}
	return state
}

func (r *defaultPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := r.write(&plan, &resp.Diagnostics)
	if state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *defaultPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	live, err := r.apiClient.GetScmPostureDefaultPolicy()
	if err != nil {
		resp.Diagnostics.AddError(errReadDefaultPolicy, err.Error())
		return
	}
	newState, err := apiToState(live)
	if err != nil {
		resp.Diagnostics.AddError(errReadDefaultPolicy, err.Error())
		return
	}
	if state.Controls == nil && newState.Controls != nil {
		newState.Controls = nil
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *defaultPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state := r.write(&plan, &resp.Diagnostics)
	if state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *defaultPolicyResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing the SCM posture default policy from state; the built-in policy itself cannot be deleted and is left untouched.")
}

func controlsToAPI(controls []controlModel) []api_client.ScmPostureControlOverride {
	out := make([]api_client.ScmPostureControlOverride, 0, len(controls))
	for _, c := range controls {
		item := api_client.ScmPostureControlOverride{
			ID:       c.ID.ValueString(),
			Priority: c.Priority.ValueString(),
		}
		if !c.Disabled.IsNull() && !c.Disabled.IsUnknown() {
			v := c.Disabled.ValueBool()
			item.Disabled = &v
		}
		out = append(out, item)
	}
	return out
}

// decodePolicyData parses the live policy_data blob. A decode failure must be
// surfaced, never swallowed: callers echo the decoded controls back in a
// full-replacement PUT, so treating malformed JSON as "no controls" would erase
// every live override.
func decodePolicyData(p *api_client.ScmPostureDefaultPolicy) (api_client.ScmPostureDefaultPolicyData, error) {
	var data api_client.ScmPostureDefaultPolicyData
	if len(p.PolicyData) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(p.PolicyData, &data); err != nil {
		return data, fmt.Errorf("could not decode live policy_data: %w", err)
	}
	return data, nil
}

func liveControls(p *api_client.ScmPostureDefaultPolicy) ([]api_client.ScmPostureControlOverride, error) {
	data, err := decodePolicyData(p)
	if err != nil {
		return nil, err
	}
	if data.Controls == nil {
		return []api_client.ScmPostureControlOverride{}, nil
	}
	return data.Controls, nil
}

func apiToState(p *api_client.ScmPostureDefaultPolicy) (*resourceModel, error) {
	state := &resourceModel{
		ID:          types.StringValue(p.ID),
		Name:        types.StringValue(p.Name),
		Description: types.StringValue(p.Description),
		Disabled:    types.BoolValue(p.Disabled),
	}
	data, err := decodePolicyData(p)
	if err != nil {
		return nil, err
	}
	if len(data.Controls) > 0 {
		state.Controls = make([]controlModel, 0, len(data.Controls))
		for _, c := range data.Controls {
			m := controlModel{ID: types.StringValue(c.ID)}
			if c.Disabled != nil {
				m.Disabled = types.BoolValue(*c.Disabled)
			}
			if c.Priority != "" {
				m.Priority = types.StringValue(c.Priority)
			}
			state.Controls = append(state.Controls, m)
		}
	}
	return state, nil
}
