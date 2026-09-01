package crown_jewel

import (
	"context"
	"fmt"
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

// nonWhitespaceRegex rejects whitespace-only Reason values at plan time.
var nonWhitespaceRegex = regexp.MustCompile(`.*\S.*`)

var (
	_ resource.Resource                = &crownJewelResource{}
	_ resource.ResourceWithConfigure   = &crownJewelResource{}
	_ resource.ResourceWithImportState = &crownJewelResource{}
)

const errCreatingCrownJewel = "Error creating crown jewel"

type crownJewelResource struct {
	apiClient *api_client.APIClient
}

// stateModel is shared with the data source (id / group_unique_id / description).
type stateModel struct {
	ID            types.String `tfsdk:"id"`
	GroupUniqueID types.String `tfsdk:"group_unique_id"`
	Description   types.String `tfsdk:"description"`
}

type resourceModel struct {
	ID            types.String   `tfsdk:"id"`
	GroupUniqueID types.String   `tfsdk:"group_unique_id"`
	Description   types.String   `tfsdk:"description"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
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

func (r *crownJewelResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Marks an asset as a user-defined crown jewel, matching the Orca UI (Mark as Crown Jewel). " +
			"The asset is identified by `group_unique_id` and must exist in inventory. Create fails if the asset " +
			"is already user-marked — import first to adopt it. Orca-detected assets can still be marked. " +
			"Create also needs permission to query inventory (`POST /api/serving-layer/query`). " +
			"Update changes the Reason on a mark this resource already manages. Destroy matches the UI disable action.",
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
					"Create requires the id to exist in inventory and not already be user-marked.",
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
					"(\"Mark as Crown Jewel\"). UI presets are `Critical business function`, `Customer data`, " +
					"`High blast radius`, or Other (free text, max 50 characters).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(nonWhitespaceRegex, "must contain at least one non-whitespace character"),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// applyPlan POSTs the planned crown jewel and writes computed fields back onto plan.
func (r *crownJewelResource) applyPlan(plan *resourceModel, timeout time.Duration) error {
	if _, err := r.apiClient.SetCrownJewel(plan.GroupUniqueID.ValueString(), plan.Description.ValueString(), timeout); err != nil {
		return err
	}
	// Keep Required attributes from the plan; only refresh Computed ones from the API.
	plan.ID = plan.GroupUniqueID
	return nil
}

func (r *crownJewelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, api_client.DefaultCrownJewelTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	gid := plan.GroupUniqueID.ValueString()
	existing, err := r.apiClient.GetCrownJewel(gid)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingCrownJewel,
			fmt.Sprintf("Could not look up existing crown jewel %s: %s", gid, err.Error()),
		)
		return
	}
	if existing != nil {
		resp.Diagnostics.AddError(
			"Crown jewel already exists",
			fmt.Sprintf("Asset %q is already a user-marked crown jewel. The Orca UI does not offer Mark on an already-marked asset. Import it instead: terraform import orcasecurity_crown_jewel.<name> %s", gid, gid),
		)
		return
	}
	exists, err := r.apiClient.InventoryGroupExists(gid)
	if err != nil {
		resp.Diagnostics.AddError(
			errCreatingCrownJewel,
			fmt.Sprintf("Could not verify inventory asset %s: %s", gid, err.Error()),
		)
		return
	}
	if !exists {
		resp.Diagnostics.AddError(
			"Asset not found",
			fmt.Sprintf("No inventory asset found for group_unique_id %q. Crown jewels can only be set on existing assets.", gid),
		)
		return
	}

	if err := r.applyPlan(&plan, timeout); err != nil {
		resp.Diagnostics.AddError(
			errCreatingCrownJewel,
			"Could not create crown jewel, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *crownJewelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
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
	var plan resourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, api_client.DefaultCrownJewelTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyPlan(&plan, timeout); err != nil {
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
	var state resourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, api_client.DefaultCrownJewelTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCrownJewel(state.ID.ValueString(), timeout)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting crown jewel",
			"Could not delete crown jewel, unexpected error: "+err.Error(),
		)
		return
	}
}
