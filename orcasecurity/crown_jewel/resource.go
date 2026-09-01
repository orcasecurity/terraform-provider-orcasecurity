package crown_jewel

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource                = &crownJewelResource{}
	_ resource.ResourceWithConfigure   = &crownJewelResource{}
	_ resource.ResourceWithImportState = &crownJewelResource{}
)

type crownJewelResource struct {
	apiClient *api_client.APIClient
}

type stateModel struct {
	ID            types.String `tfsdk:"id"`
	GroupUniqueID types.String `tfsdk:"group_unique_id"`
	Description   types.String `tfsdk:"description"`
}

func NewCrownJewelResource() resource.Resource {
	return &crownJewelResource{}
}

func (r *crownJewelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crown_jewel"
}

func (r *crownJewelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *crownJewelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *crownJewelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Marks an asset as a user-defined crown jewel. The asset is identified by `group_unique_id`. " +
			"Create and update both upsert: applying this resource on an asset that is already user-marked " +
			"overwrites the existing reason instead of failing. Prefer `terraform import` to adopt an " +
			"existing mark without changing it on first apply. Orca-detected crown jewels are engine-managed " +
			"and cannot be deleted through this resource; applying here adds a user-marked overlay (hybrid).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Same as `group_unique_id`.",
			},
			"group_unique_id": schema.StringAttribute{
				Description: "Inventory group unique id of the asset to mark as a crown jewel. Changing this value replaces the resource. " +
					"If the asset is already user-marked, apply updates that mark (upsert) rather than creating a second one.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Reason for marking the asset as a crown jewel — the same field as **Reason** in the Orca UI " +
					"(\"Mark as Crown Jewel\"). Common UI values are `Critical business function`, `Customer data`, " +
					"`High blast radius`, or free text when choosing Other. Required: the API accepts omit, but " +
					"creating without a reason stores a null description and breaks list/read of crown jewels.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

// applyPlan POSTs the planned crown jewel and writes computed fields back onto plan.
// Create and Update share this path because the API is a single upsert.
func (r *crownJewelResource) applyPlan(plan *stateModel) error {
	if _, err := r.apiClient.SetCrownJewel(plan.GroupUniqueID.ValueString(), plan.Description.ValueString()); err != nil {
		return err
	}
	// Keep Required attributes from the plan; only refresh Computed ones from the API.
	plan.ID = plan.GroupUniqueID
	return nil
}

func (r *crownJewelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyPlan(&plan); err != nil {
		resp.Diagnostics.AddError(
			"Error creating crown jewel",
			"Could not create crown jewel, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *crownJewelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	instance, err := r.apiClient.GetCrownJewel(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading crown jewel",
			fmt.Sprintf("Could not read crown jewel %s: %s", id, err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Crown jewel %s is missing on the remote side.", id))
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(instance.GroupUniqueID)
	state.GroupUniqueID = types.StringValue(instance.GroupUniqueID)
	state.Description = types.StringValue(instance.Description)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *crownJewelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyPlan(&plan); err != nil {
		resp.Diagnostics.AddError(
			"Error updating crown jewel",
			"Could not update crown jewel, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *crownJewelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCrownJewel(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting crown jewel",
			"Could not delete crown jewel, unexpected error: "+err.Error(),
		)
		return
	}
}
