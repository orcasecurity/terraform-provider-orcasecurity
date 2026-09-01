package custom_compliance_framework

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/integrations_common"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

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

const errReadingFramework = "Error reading custom compliance framework"

var (
	_ resource.Resource                   = &customComplianceFrameworkResource{}
	_ resource.ResourceWithConfigure      = &customComplianceFrameworkResource{}
	_ resource.ResourceWithImportState    = &customComplianceFrameworkResource{}
	_ resource.ResourceWithValidateConfig = &customComplianceFrameworkResource{}
	_ resource.ResourceWithModifyPlan     = &customComplianceFrameworkResource{}
)

type customComplianceFrameworkResource struct {
	apiClient *api_client.APIClient
}

type customComplianceFrameworkResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Visibility         types.String `tfsdk:"visibility"`
	Scope              types.String `tfsdk:"scope"`
	ForcedCloudVendors types.Set    `tfsdk:"forced_cloud_vendors"`
	Sections           types.List   `tfsdk:"sections"`
}

func NewCustomComplianceFrameworkResource() resource.Resource {
	return &customComplianceFrameworkResource{}
}

func (r *customComplianceFrameworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_compliance_framework"
}

func (r *customComplianceFrameworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *customComplianceFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func testAttributes() map[string]schema.Attribute {
	computedOptional := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: desc,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		}
	}
	return map[string]schema.Attribute{
		"rule_id": schema.StringAttribute{
			Required:    true,
			Description: "The rule ID for the test/control.",
		},
		"rule_id_in_framework": computedOptional(
			"The identifier for this rule within the framework (e.g. `1.1`, `1.1.1`). " +
				"Omitted values are derived as `<section_id_in_framework>.<1-based index>` " +
				"within this resource's own section tree (e.g. `1.1`), matching the Orca UI — " +
				"not from a source framework's catalog ids. Changing the section id re-derives " +
				"an omitted value. On read this is the catalog `reference_id`.",
		),
		"priority":            computedOptional("Control priority as accepted by the API (e.g. `Medium`)."),
		"control_unique_id":   computedOptional("Catalog control unique id. Echoed when the API returns it."),
		"origin_framework_id": computedOptional("Origin framework id when this control was copied from another framework."),
	}
}

func sectionAttributes(remainingDepth int) map[string]schema.Attribute {
	if remainingDepth == 0 {
		return map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Section name. This nesting level is not supported; ValidateConfig rejects it.",
			},
		}
	}
	attrs := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Section name.",
		},
		"section_id_in_framework": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Description: "Sets this section's id, which becomes the prefix of each control's " +
				"`rule_id_in_framework` (`7` → `7.1`, `7.2`). Updatable — changing it re-derives " +
				"omitted control ids. Must be an unsigned integer, unique and strictly ascending " +
				"among siblings, and a nested section must extend its parent (`7.2`) — the API " +
				"returns sections sorted by id. Omitted values take the next integer above the " +
				"previous sibling (`7.2` then omitted → `7.3`); an id already assigned is kept " +
				"until you change it. On read this is the catalog section `id`.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"tests": schema.ListNestedAttribute{
			Optional:    true,
			Description: "Tests (controls) within this section. A section may have tests or sub-sections, never both. Omit the attribute rather than setting `tests = []` — an empty tests list is read back as null.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: testAttributes(),
			},
		},
	}
	if remainingDepth > 0 {
		attrs["sections"] = schema.ListNestedAttribute{
			Optional:    true,
			Description: nestedSectionsAttrDescription(remainingDepth),
			NestedObject: schema.NestedAttributeObject{
				Attributes: sectionAttributes(remainingDepth - 1),
			},
		}
	}
	return attrs
}

func (r *customComplianceFrameworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a custom compliance framework resource. Sections are read back " +
			"from GET /api/compliance/catalog/{id}, so import and drift detection cover the tree. " +
			"A section may contain tests or nested sections, never both — the API would otherwise " +
			"silently flatten it. Nesting is at most three levels (an API limit); a fourth nested " +
			"`sections` block is rejected in ValidateConfig. " +
			"Drafts (`/api/compliance/frameworks/drafts` and `draft_id` on create) are a UI-only " +
			"workflow and are not managed by this resource. " +
			"Omit `scope` to create the framework inactive; ongoing activation belongs to " +
			"`orcasecurity_compliance_framework_selection`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Framework ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Framework name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Framework description.",
			},
			"visibility": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Who can see the framework: `Organizational` or `Personal`. The server default is used when omitted. " +
					"`Personal` can be promoted to `Organizational`; the reverse is rejected by the API. " +
					"`Personal` cannot be combined with `scope = \"organization\"` (the API returns 400). " +
					"Personal frameworks are visible only to the creating user; a different API token sees 404 and Terraform will try to recreate them.",
				Validators: []validator.String{
					stringvalidator.OneOf(api_client.VisibilityOrganizational, api_client.VisibilityPersonal),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Optional: true,
				Description: "Create-only activation: `user` or `organization`. PUT ignores this field. " +
					"Omitting it leaves the new framework inactive (`selection_scopes: []`). " +
					"`visibility = \"Personal\"` cannot use `organization`. " +
					"Ongoing enable/disable belongs to `orcasecurity_compliance_framework_selection`.",
				Validators: []validator.String{
					stringvalidator.OneOf(api_client.ScopeUser, api_client.ScopeOrganization),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"forced_cloud_vendors": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Force the framework onto these cloud vendors. Sent only when non-empty. " +
					"`forced_cloud_vendors = []` is treated as omit: the provider does not send the key, " +
					"so enforcement is cleared on update (the API 400s on an explicit empty list). " +
					"Omitting the attribute on update also clears enforcement.",
			},
			"sections": schema.ListNestedAttribute{
				Required:    true,
				Description: "Framework sections containing tests/controls. Read from the catalog; order is preserved. Nested at most three levels (an API limit). Every leaf section must have at least one test.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: sectionAttributes(schemaSectionDepth - 1),
				},
			},
		},
	}
}

func (r *customComplianceFrameworkResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateSections(resp, config.Sections, path.Root("sections"))
	validatePersonalOrganization(resp, config)
}

func validatePersonalOrganization(resp *resource.ValidateConfigResponse, config customComplianceFrameworkResourceModel) {
	if config.Visibility.IsNull() || config.Visibility.IsUnknown() || config.Scope.IsNull() || config.Scope.IsUnknown() {
		return
	}
	vis := config.Visibility.ValueString()
	if api_client.PersonalRejectsOrganization(&vis, []string{config.Scope.ValueString()}) {
		resp.Diagnostics.AddAttributeError(path.Root("scope"), api_client.ErrPersonalOrgSummary, api_client.ErrPersonalOrgDetail)
	}
}

func (r *customComplianceFrameworkResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Live non-destroy plans always have Config. Unit tests that only
	// exercise the visibility check omit it; skip the rewrite in that case.
	if req.Config.Schema != nil && !req.Config.Raw.IsNull() {
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	var state customComplianceFrameworkResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !config.Sections.IsNull() && !config.Sections.IsUnknown() && !plan.Sections.IsNull() && !plan.Sections.IsUnknown() {
		sections, d := rewriteSectionsPlan(planRewrite{
			config: config.Sections,
			plan:   plan.Sections,
			state:  state.Sections,
		}, schemaSectionDepth-1, false)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Sections = sections
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if req.State.Raw.IsNull() {
		return
	}
	if state.Visibility.IsNull() || state.Visibility.IsUnknown() || plan.Visibility.IsNull() || plan.Visibility.IsUnknown() {
		return
	}
	if api_client.VisibilityDowngrade(state.Visibility.ValueString(), plan.Visibility.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("visibility"), api_client.ErrVisibilityDowngradeSummary, api_client.ErrVisibilityDowngradeDetail)
	}
}

func (r *customComplianceFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := requestFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateCustomComplianceFramework(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom compliance framework",
			"Could not create custom compliance framework, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID.String())
	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customComplianceFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ok, d := r.populate(ctx, &state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !ok {
		tflog.Warn(ctx, fmt.Sprintf("Custom compliance framework %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customComplianceFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := requestFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.apiClient.UpdateCustomComplianceFramework(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom compliance framework",
			"Could not update custom compliance framework, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.refresh(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customComplianceFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteCustomComplianceFramework(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom compliance framework",
			"Could not delete custom compliance framework, unexpected error: "+err.Error(),
		)
	}
}

func (r *customComplianceFrameworkResource) refresh(ctx context.Context, model *customComplianceFrameworkResourceModel) diag.Diagnostics {
	ok, d := r.populate(ctx, model)
	if d.HasError() {
		return d
	}
	if !ok {
		var diags diag.Diagnostics
		diags.AddError(errReadingFramework, fmt.Sprintf("custom compliance framework %s disappeared after write", model.ID.ValueString()))
		return diags
	}
	return nil
}

// populate reads metadata + catalog into model. ok=false means 404/gone.
func (r *customComplianceFrameworkResource) populate(ctx context.Context, model *customComplianceFrameworkResourceModel) (bool, diag.Diagnostics) {
	id := model.ID.ValueString()
	fw, err := r.apiClient.GetCustomComplianceFramework(id)
	if err != nil {
		var d diag.Diagnostics
		d.AddError(errReadingFramework, err.Error())
		return false, d
	}
	if fw == nil {
		return false, nil
	}

	model.Name = types.StringValue(fw.DisplayName)
	model.Description = tfconv.StringPtrOrNull(fw.Description)
	model.Visibility = tfconv.StringPtrOrNull(fw.Visibility)

	forced := []string(nil)
	if fw.IsForcedCloudVendors != nil && *fw.IsForcedCloudVendors {
		forced = fw.FrameworkCloudVendors
	}
	set, d := integrations_common.OptionalSetMatchPlan(ctx, model.ForcedCloudVendors, forced)
	if d.HasError() {
		return false, d
	}
	model.ForcedCloudVendors = set

	catalog, err := r.apiClient.GetComplianceCatalogFramework(id)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(errReadingFramework, err.Error())
		return false, diags
	}
	if catalog == nil {
		return false, catalogMissingDiag(id)
	}
	sections, d := sectionsFromCatalog(catalog.Sections, schemaSectionDepth-1)
	if d.HasError() {
		return false, d
	}
	model.Sections = sections
	return true, nil
}

func requestFromPlan(ctx context.Context, plan customComplianceFrameworkResourceModel) (api_client.CustomComplianceFrameworkRequest, diag.Diagnostics) {
	vendors, d := integrations_common.StringSliceFromSet(ctx, plan.ForcedCloudVendors)
	if d.HasError() {
		return api_client.CustomComplianceFrameworkRequest{}, d
	}
	if len(vendors) == 0 {
		vendors = nil
	}
	return api_client.CustomComplianceFrameworkRequest{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		Visibility:         plan.Visibility.ValueString(),
		Scope:              plan.Scope.ValueString(),
		ForcedCloudVendors: vendors,
		Sections:           sectionsToAPI(plan.Sections),
	}, d
}
