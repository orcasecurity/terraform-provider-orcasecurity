package compliance_framework_selection

import (
	"context"
	"fmt"
	"sort"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                = &complianceFrameworkSelectionResource{}
	_ resource.ResourceWithConfigure   = &complianceFrameworkSelectionResource{}
	_ resource.ResourceWithImportState = &complianceFrameworkSelectionResource{}
)

const (
	errReadingSelection  = "Error reading compliance framework selection"
	errUpdatingSelection = "Error updating compliance framework selection"
)

type complianceFrameworkSelectionResource struct {
	apiClient *api_client.APIClient
}

type resourceModel struct {
	ID          types.String `tfsdk:"id"`
	FrameworkID types.String `tfsdk:"framework_id"`
	Scopes      types.Set    `tfsdk:"scopes"`
	DisplayName types.String `tfsdk:"display_name"`
	Custom      types.Bool   `tfsdk:"custom"`
	IsReady     types.Bool   `tfsdk:"is_ready"`
}

func NewComplianceFrameworkSelectionResource() resource.Resource {
	return &complianceFrameworkSelectionResource{}
}

func (r *complianceFrameworkSelectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_framework_selection"
}

func (r *complianceFrameworkSelectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *complianceFrameworkSelectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("framework_id"), req, resp)
}

func (r *complianceFrameworkSelectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages which scopes (`user`, `organization`) a compliance framework is " +
			"selected for. One resource per framework. `scopes = []` is the explicit disable " +
			"action — it DELETEs every held scope. " +
			"**Destroy is state-only** and does not deselect the framework: built-in " +
			"frameworks exist before Terraform and the `organization` scope is shared tenant " +
			"state. This resource never deletes the framework itself. " +
			"The `user` scope is token-scoped (the API token's own user), so a different token " +
			"sees a different `user` selection. Two resources pointing at the same " +
			"`framework_id` will fight.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as `framework_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"framework_id": schema.StringAttribute{
				Required:    true,
				Description: "System or custom framework id. Changing this value replaces the resource. Also the import id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"scopes": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Selection scopes to hold. Valid values: `user`, `organization`. " +
					"An empty set disables the framework (DELETE of every held scope). " +
					"A set, not a list — the API returns them unordered. " +
					"Personal frameworks cannot hold `organization` (the API returns 400).",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(api_client.ScopeUser, api_client.ScopeOrganization)),
				},
			},
			"display_name": schema.StringAttribute{
				Computed:    true,
				Description: "Framework display name from the select map.",
			},
			"custom": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is a custom (organization-created) framework.",
			},
			"is_ready": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the framework is ready. Null when the select entry omits the field.",
			},
		},
	}
}

func (r *complianceFrameworkSelectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.lookup(plan.FrameworkID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating compliance framework selection",
			"Could not read current selection: "+err.Error(),
		)
		return
	}
	if entry == nil {
		resp.Diagnostics.AddError(
			"Compliance framework not found",
			fmt.Sprintf("Framework %q does not exist, so it cannot be selected.", plan.FrameworkID.ValueString()),
		)
		return
	}

	desired, diags := integrations_common.StringSliceFromSet(ctx, plan.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d := personalOrganizationDiag(entry.Visibility, desired); d != nil {
		resp.Diagnostics.Append(d...)
		return
	}

	if err := r.applyScopeDiff(plan.FrameworkID.ValueString(), entry.SelectionScopes, desired); err != nil {
		resp.Diagnostics.AddError(
			"Error creating compliance framework selection",
			"Could not apply scopes, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *complianceFrameworkSelectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, err := r.lookup(state.FrameworkID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingSelection,
			fmt.Sprintf("Could not read framework %s: %s", state.FrameworkID.ValueString(), err.Error()),
		)
		return
	}
	if entry == nil {
		tflog.Warn(ctx, fmt.Sprintf("Compliance framework %s is missing on the remote side.", state.FrameworkID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(populateFromEntry(ctx, &state, entry)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *complianceFrameworkSelectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	var state resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	from, d := integrations_common.StringSliceFromSet(ctx, state.Scopes)
	resp.Diagnostics.Append(d...)
	to, d := integrations_common.StringSliceFromSet(ctx, plan.Scopes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if containsScope(to, api_client.ScopeOrganization) {
		entry, err := r.lookup(plan.FrameworkID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(errUpdatingSelection, "Could not read current selection: "+err.Error())
			return
		}
		if entry != nil {
			if pd := personalOrganizationDiag(entry.Visibility, to); pd != nil {
				resp.Diagnostics.Append(pd...)
				return
			}
		}
	}

	if err := r.applyScopeDiff(plan.FrameworkID.ValueString(), from, to); err != nil {
		resp.Diagnostics.AddError(errUpdatingSelection, "Could not update scopes, unexpected error: "+err.Error())
		return
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *complianceFrameworkSelectionResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentionally empty: destroy is state-only. This resource never created
	// the framework, and the organization scope is shared tenant state.
	// Disabling is `scopes = []`.
}

func personalOrganizationDiag(visibility *string, scopes []string) diag.Diagnostics {
	if !api_client.PersonalRejectsOrganization(visibility, scopes) {
		return nil
	}
	var d diag.Diagnostics
	d.AddError(api_client.ErrPersonalOrgSummary, api_client.ErrPersonalOrgDetail)
	return d
}

func containsScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

func (r *complianceFrameworkSelectionResource) lookup(id string) (*api_client.ComplianceFramework, error) {
	all, err := r.apiClient.GetComplianceFrameworkSelections()
	if err != nil {
		return nil, err
	}
	entry, ok := all[id]
	if !ok {
		return nil, nil
	}
	return &entry, nil
}

func (r *complianceFrameworkSelectionResource) refresh(ctx context.Context, plan *resourceModel) diag.Diagnostics {
	entry, err := r.lookup(plan.FrameworkID.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError(errReadingSelection, err.Error())
		return d
	}
	if entry == nil {
		var d diag.Diagnostics
		d.AddError(errReadingSelection, fmt.Sprintf("framework %s disappeared after write", plan.FrameworkID.ValueString()))
		return d
	}
	return populateFromEntry(ctx, plan, entry)
}

func populateFromEntry(ctx context.Context, model *resourceModel, entry *api_client.ComplianceFramework) diag.Diagnostics {
	scopes, d := tfconv.StringSetFromAPIPreserveNull(ctx, model.Scopes, entry.SelectionScopes)
	if d.HasError() {
		return d
	}
	model.ID = types.StringValue(entry.ID)
	model.FrameworkID = types.StringValue(entry.ID)
	model.Scopes = scopes
	model.DisplayName = types.StringValue(entry.DisplayName)
	model.Custom = types.BoolValue(entry.Custom)
	model.IsReady = tfconv.BoolPtrOrNull(entry.IsReady)
	return d
}

func (r *complianceFrameworkSelectionResource) applyScopeDiff(frameworkID string, from, to []string) error {
	add, remove := DiffScopes(from, to)
	for _, scope := range add {
		if err := r.apiClient.SelectComplianceFrameworks([]string{frameworkID}, scope); err != nil {
			return err
		}
	}
	for _, scope := range remove {
		if err := r.apiClient.DeselectComplianceFrameworks([]string{frameworkID}, scope); err != nil {
			return err
		}
	}
	return nil
}

// DiffScopes returns the scopes to POST (add) and DELETE (remove) to go from
// `from` to `to`. Order is sorted so callers and tests see a stable sequence.
func DiffScopes(from, to []string) (add, remove []string) {
	fromSet := map[string]struct{}{}
	toSet := map[string]struct{}{}
	for _, s := range from {
		fromSet[s] = struct{}{}
	}
	for _, s := range to {
		toSet[s] = struct{}{}
		if _, ok := fromSet[s]; !ok {
			add = append(add, s)
		}
	}
	for _, s := range from {
		if _, ok := toSet[s]; !ok {
			remove = append(remove, s)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	return add, remove
}
