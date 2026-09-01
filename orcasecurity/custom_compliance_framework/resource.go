package custom_compliance_framework

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
)

type customComplianceFrameworkResource struct {
	apiClient *api_client.APIClient
}

type customComplianceFrameworkResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	Name               types.String   `tfsdk:"name"`
	Description        types.String   `tfsdk:"description"`
	Visibility         types.String   `tfsdk:"visibility"`
	Scope              types.String   `tfsdk:"scope"`
	ForcedCloudVendors types.Set      `tfsdk:"forced_cloud_vendors"`
	Sections           []sectionModel `tfsdk:"sections"`
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
				"Omitted values are derived as `<section-id>.<1-based index>`, matching the Orca UI. " +
				"On read this is the catalog `reference_id`.",
		),
		"priority":            computedOptional("Control priority as accepted by the API (e.g. `Medium`)."),
		"control_unique_id":   computedOptional("Catalog control unique id. Echoed when the API returns it."),
		"origin_framework_id": computedOptional("Origin framework id when this control was copied from another framework."),
		"reference_id":        computedOptional("Server-echoed control id. Same value as `rule_id_in_framework` after apply."),
	}
}

func sectionAttributes(remainingDepth int) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Section name.",
		},
		"tests": schema.ListNestedAttribute{
			Optional:    true,
			Description: "Tests (controls) within this section. A section may have tests or sub-sections, never both.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: testAttributes(),
			},
		},
	}
	if remainingDepth > 0 {
		attrs["sections"] = schema.ListNestedAttribute{
			Optional: true,
			Description: "Nested sub-sections. The API stores exactly three levels " +
				"(sections → sections → sections); a fourth is a config error because the " +
				"server would drop it and reparent its controls. A section may have tests or sub-sections, never both.",
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
			"silently flatten it. Nesting is at most three levels; a fourth is rejected because " +
			"the API would drop it and reparent its controls. " +
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
				Optional:    true,
				Computed:    true,
				Description: "Who can see the framework: `Organizational` or `Personal`. The server default is used when omitted.",
				Validators: []validator.String{
					stringvalidator.OneOf("Organizational", "Personal"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				Optional: true,
				Description: "Create-only activation: `user` or `organization`. PUT ignores this field. " +
					"Omitting it leaves the new framework inactive (`selection_scopes: []`). " +
					"Ongoing enable/disable belongs to `orcasecurity_compliance_framework_selection`.",
				Validators: []validator.String{
					stringvalidator.OneOf("user", "organization"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"forced_cloud_vendors": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Force the framework onto these cloud vendors. Sent only when non-empty; " +
					"omitting the attribute on update clears enforcement (API behavior). " +
					"An explicit empty list is rejected by the API.",
			},
			"sections": schema.ListNestedAttribute{
				Required:    true,
				Description: "Framework sections containing tests/controls. Read from the catalog; order is preserved. Nested at most three levels (an API limit).",
				NestedObject: schema.NestedAttributeObject{
					// remainingDepth 3 exposes a fourth nested `sections` so ValidateConfig
					// can reject it with a data-loss message instead of an opaque schema error.
					Attributes: sectionAttributes(3),
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
	validateSections(resp, config.Sections)
}

func rejectMixedSection(resp *resource.ValidateConfigResponse, p path.Path, tests, children int) {
	if sectionHasTestsAndChildren(tests, children) {
		resp.Diagnostics.AddAttributeError(p, invalidSectionSummary, mixedSectionError(p.String()))
	}
}

func validateLeafSections(resp *resource.ValidateConfigResponse, parent path.Path, leaves []leafSectionModel) {
	for k, leaf := range leaves {
		lp := parent.AtName("sections").AtListIndex(k)
		rejectMixedSection(resp, lp, len(leaf.Tests), len(leaf.Sections))
		if len(leaf.Sections) > 0 {
			resp.Diagnostics.AddAttributeError(lp.AtName("sections"), "Section nesting too deep", depthSectionMessage)
		}
	}
}

func validateMidSections(resp *resource.ValidateConfigResponse, parent path.Path, mids []midSectionModel) {
	for j, mid := range mids {
		mp := parent.AtName("sections").AtListIndex(j)
		rejectMixedSection(resp, mp, len(mid.Tests), len(mid.Sections))
		validateLeafSections(resp, mp, mid.Sections)
	}
}

func validateSections(resp *resource.ValidateConfigResponse, sections []sectionModel) {
	for i, s := range sections {
		p := path.Root("sections").AtListIndex(i)
		rejectMixedSection(resp, p, len(s.Tests), len(s.Sections))
		validateMidSections(resp, p, s.Sections)
	}
}

func (r *customComplianceFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customComplianceFrameworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := createRequestFromPlan(ctx, plan)
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
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(errReadingFramework, err.Error())
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

	ok, err := r.populate(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError(
			errReadingFramework,
			fmt.Sprintf("Could not read custom compliance framework ID %s: %s", state.ID.ValueString(), err.Error()),
		)
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

	updateReq, diags := updateRequestFromPlan(ctx, plan)
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

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(errReadingFramework, err.Error())
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

func (r *customComplianceFrameworkResource) refresh(ctx context.Context, model *customComplianceFrameworkResourceModel) error {
	ok, err := r.populate(ctx, model)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("custom compliance framework %s disappeared after write", model.ID.ValueString())
	}
	return nil
}

// populate reads metadata + catalog into model. ok=false means 404/gone.
func (r *customComplianceFrameworkResource) populate(ctx context.Context, model *customComplianceFrameworkResourceModel) (bool, error) {
	id := model.ID.ValueString()
	fw, err := r.apiClient.GetCustomComplianceFramework(id)
	if err != nil {
		return false, err
	}
	if fw == nil {
		return false, nil
	}

	model.Name = types.StringValue(fw.DisplayName)
	if fw.Description == nil || *fw.Description == "" {
		if model.Description.IsNull() || model.Description.IsUnknown() {
			model.Description = types.StringNull()
		}
	} else {
		model.Description = types.StringValue(*fw.Description)
	}
	if fw.Visibility == nil || *fw.Visibility == "" {
		model.Visibility = types.StringNull()
	} else {
		model.Visibility = types.StringValue(*fw.Visibility)
	}

	forced := []string(nil)
	if fw.IsForcedCloudVendors != nil && *fw.IsForcedCloudVendors {
		forced = fw.FrameworkCloudVendors
	}
	set, d := integrations_common.OptionalSetMatchPlan(ctx, model.ForcedCloudVendors, forced)
	if d.HasError() {
		return false, fmt.Errorf("%s", d.Errors())
	}
	model.ForcedCloudVendors = set

	catalog, err := r.apiClient.GetComplianceCatalogFramework(id)
	if err != nil {
		return false, err
	}
	if catalog != nil {
		model.Sections = sectionsFromCatalog(catalog.Sections)
	}
	return true, nil
}

func createRequestFromPlan(ctx context.Context, plan customComplianceFrameworkResourceModel) (api_client.CustomComplianceFrameworkCreateRequest, diag.Diagnostics) {
	req, d := updateRequestFromPlan(ctx, plan)
	if d.HasError() {
		return api_client.CustomComplianceFrameworkCreateRequest{}, d
	}
	return api_client.CustomComplianceFrameworkCreateRequest{
		Name:               req.Name,
		Description:        req.Description,
		Visibility:         req.Visibility,
		Scope:              plan.Scope.ValueString(),
		ForcedCloudVendors: req.ForcedCloudVendors,
		Sections:           req.Sections,
	}, d
}

func updateRequestFromPlan(ctx context.Context, plan customComplianceFrameworkResourceModel) (api_client.CustomComplianceFrameworkUpdateRequest, diag.Diagnostics) {
	vendors, d := integrations_common.StringSliceFromSet(ctx, plan.ForcedCloudVendors)
	if d.HasError() {
		return api_client.CustomComplianceFrameworkUpdateRequest{}, d
	}
	if len(vendors) == 0 {
		vendors = nil
	}
	return api_client.CustomComplianceFrameworkUpdateRequest{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		Visibility:         plan.Visibility.ValueString(),
		ForcedCloudVendors: vendors,
		Sections:           sectionsToAPI(plan.Sections),
	}, d
}
