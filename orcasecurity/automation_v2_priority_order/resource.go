package automation_v2_priority_order

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &automationPriorityOrderResource{}
	_ resource.ResourceWithConfigure = &automationPriorityOrderResource{}
)

// resourceID is the fixed singleton ID: priority is one global ordering per
// organization, so only one instance of this resource is meaningful.
const resourceID = "automation_priority_order"

type automationPriorityOrderResource struct {
	apiClient *api_client.APIClient
}

type automationPriorityOrderResourceModel struct {
	ID            types.String `tfsdk:"id"`
	AutomationIDs types.List   `tfsdk:"automation_ids"`
}

func NewAutomationPriorityOrderResource() resource.Resource {
	return &automationPriorityOrderResource{}
}

func (r *automationPriorityOrderResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_automation_v2_priority_order"
}

func (r *automationPriorityOrderResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *automationPriorityOrderResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Owns the top evaluation-order positions of the organization's automations. " +
			"The listed automations are assigned priorities 1..N in list order on every apply. " +
			"Declare at most one instance, and do not combine it with the `priority` attribute " +
			"on `orcasecurity_automation_v2` resources — the two owners will fight. " +
			"Requires a token with the global Rules Create (admin) permission.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"automation_ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Automation IDs in desired evaluation order; the first entry gets priority 1. " +
					"Automations not listed keep their relative order below the listed ones.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
				},
			},
		},
	}
}

// assertOrder sets each automation's priority to its 1-based list position,
// sequentially and unconditionally. Sequential single-writer application makes
// the outcome deterministic; the server no-ops moves where old == new, and a
// pre-read skip would go stale because each move shifts later automations.
func (r *automationPriorityOrderResource) assertOrder(ids []string) error {
	for i, id := range ids {
		if _, err := r.apiClient.SetAutomationV2Priority(id, int64(i+1)); err != nil {
			return fmt.Errorf("setting priority %d for automation %s: %w", i+1, id, err)
		}
	}
	return nil
}

// topNIDs returns the IDs of the first n automations in server evaluation
// order (fewer if the organization has fewer automations).
func (r *automationPriorityOrderResource) topNIDs(n int) ([]string, error) {
	instances, err := r.apiClient.ListAutomationsV2()
	if err != nil {
		return nil, err
	}
	if n > len(instances) {
		n = len(instances)
	}
	ids := make([]string, 0, n)
	for _, instance := range instances[:n] {
		ids = append(ids, instance.ID)
	}
	return ids, nil
}

func (r *automationPriorityOrderResource) listToIDs(ctx context.Context, list types.List) []string {
	var ids []string
	_ = list.ElementsAs(ctx, &ids, false)
	return ids
}

func (r *automationPriorityOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationPriorityOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.assertOrder(r.listToIDs(ctx, plan.AutomationIDs)); err != nil {
		resp.Diagnostics.AddError(
			"Error setting automation priority order",
			"Could not set automation priority order: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(resourceID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *automationPriorityOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationPriorityOrderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tracked := r.listToIDs(ctx, state.AutomationIDs)
	actual, err := r.topNIDs(len(tracked))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading automation priority order",
			"Could not list automations: "+err.Error(),
		)
		return
	}

	actualList, diags := types.ListValueFrom(ctx, types.StringType, actual)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.AutomationIDs = actualList
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *automationPriorityOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationPriorityOrderResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.assertOrder(r.listToIDs(ctx, plan.AutomationIDs)); err != nil {
		resp.Diagnostics.AddError(
			"Error setting automation priority order",
			"Could not set automation priority order: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(resourceID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *automationPriorityOrderResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentionally empty: deleting this resource only stops managing the
	// ordering; automations keep their current positions, and there is
	// nothing to clean up server-side.
}
