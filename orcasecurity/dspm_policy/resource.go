package dspm_policy

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &dspmPolicyResource{}
	_ resource.ResourceWithConfigure   = &dspmPolicyResource{}
	_ resource.ResourceWithImportState = &dspmPolicyResource{}
)

type dspmPolicyResource struct {
	apiClient *api_client.APIClient
}

type documentModel struct {
	Detectors  types.List `tfsdk:"detectors"`
	Categories types.List `tfsdk:"categories"`
	Regions    types.List `tfsdk:"regions"`
	Industries types.List `tfsdk:"industries"`
	Tags       types.List `tfsdk:"tags"`
	Countries  types.List `tfsdk:"countries"`
}

type stateModel struct {
	ID              types.String   `tfsdk:"id"`
	OrganizationID  types.String   `tfsdk:"organization_id"`
	Name            types.String   `tfsdk:"name"`
	Description     types.String   `tfsdk:"description"`
	Feature         types.String   `tfsdk:"feature"`
	Tags            types.List     `tfsdk:"tags"`
	Document        *documentModel `tfsdk:"document"`
	IsDefaultPolicy types.Bool     `tfsdk:"is_default_policy"`
}

func NewDspmPolicyResource() resource.Resource {
	return &dspmPolicyResource{}
}

func (r *dspmPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_dspm_policy"
}

func (r *dspmPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *dspmPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *dspmPolicyResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides a DSPM data protection policy. A policy selects which sensitive data identifiers apply and in which context. Orca-managed default policies (`is_default_policy = true`) cannot be updated or deleted; importing one makes every change fail with a 400 error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Policy ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Orca organization ID.",
			},
			"name": schema.StringAttribute{
				Description: "Policy name. Must be unique per organization and feature. Maximum 128 characters.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(128),
				},
			},
			"description": schema.StringAttribute{
				Description: "Policy description. Maximum 512 characters.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(512),
				},
			},
			"feature": schema.StringAttribute{
				Description: "Scan feature the policy belongs to. Defaults to `DSPM Scanning`.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("DSPM Scanning"),
			},
			"tags": schema.ListAttribute{
				Description: "Policy tags.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"document": schema.SingleNestedAttribute{
				Description: "Policy selector document.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"detectors": schema.ListAttribute{
						Description: "Sensitive data identifier IDs this policy covers. Accepts built-in catalog IDs (e.g. `AUS_TAX_NUMBER`), custom identifier UUIDs, or `*` for all.",
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
					},
					"categories": schema.ListAttribute{
						Description: "Data categories. Valid values are `PII`, `PHI`, `PCI`, `SECRET`, and `OTHER`.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf("PII", "PHI", "PCI", "SECRET", "OTHER")),
						},
					},
					"regions": schema.ListAttribute{
						Description: "Region selectors. Valid values are `*`, `Europe`, `North America`, `APAC`, `LATAM`, and `MEA`.",
						Optional:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf(
								"*", "Europe", "North America", "APAC", "LATAM", "MEA",
							)),
						},
					},
					"industries": schema.ListAttribute{
						Description: "Industry selectors. Must be one of the Orca catalog industries (e.g. `Healthcare`, `Financial Services`) or `*`.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"tags": schema.ListAttribute{
						Description: "Asset tag selectors.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"countries": schema.ListAttribute{
						Description: "Country selectors. Must be one of the Orca catalog countries (e.g. `United States`, `Germany`) or `*`.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"is_default_policy": schema.BoolAttribute{
				Description: "Whether this is an Orca-managed default policy. Always false for Terraform-created policies.",
				Computed:    true,
			},
		},
	}
}

func generatePolicyPayload(ctx context.Context, plan stateModel) api_client.DSPMPolicy {
	return api_client.DSPMPolicy{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Feature:     plan.Feature.ValueString(),
		// api_client.PolicyTags serializes nil as [] — the server requires the key present.
		Tags: tfconv.StringListToAPI(ctx, plan.Tags),
		// advanced_settings is not exposed in the schema (MVP); the server expects {}
		AdvancedSettings: map[string]interface{}{},
		Document: api_client.DSPMPolicyDocument{
			SelectorDetectors:  tfconv.StringListToAPI(ctx, plan.Document.Detectors),
			SelectorCategories: tfconv.StringListToAPI(ctx, plan.Document.Categories),
			SelectorRegions:    tfconv.StringListToAPI(ctx, plan.Document.Regions),
			SelectorIndustries: tfconv.StringListToAPI(ctx, plan.Document.Industries),
			SelectorTags:       tfconv.StringListToAPI(ctx, plan.Document.Tags),
			SelectorCountries:  tfconv.StringListToAPI(ctx, plan.Document.Countries),
		},
	}
}

func (r *dspmPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateDSPMPolicy(generatePolicyPayload(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating DSPM Policy",
			"Could not create DSPM Policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.IsDefaultPolicy = types.BoolValue(instance.IsDefaultPolicy)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dspmPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetDSPMPolicy(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DSPM Policy",
			fmt.Sprintf("Could not read DSPM Policy ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("DSPM Policy %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	priorDocument := documentModel{}
	if state.Document != nil {
		priorDocument = *state.Document
	}

	tags, d := tfconv.StringListFromAPIPreserveNull(ctx, state.Tags, instance.Tags)
	resp.Diagnostics.Append(d...)
	detectors, d := types.ListValueFrom(ctx, types.StringType, instance.Document.SelectorDetectors)
	resp.Diagnostics.Append(d...)
	categories, d := tfconv.StringListFromAPIPreserveNull(ctx, priorDocument.Categories, instance.Document.SelectorCategories)
	resp.Diagnostics.Append(d...)
	regions, d := tfconv.StringListFromAPIPreserveNull(ctx, priorDocument.Regions, instance.Document.SelectorRegions)
	resp.Diagnostics.Append(d...)
	industries, d := tfconv.StringListFromAPIPreserveNull(ctx, priorDocument.Industries, instance.Document.SelectorIndustries)
	resp.Diagnostics.Append(d...)
	documentTags, d := tfconv.StringListFromAPIPreserveNull(ctx, priorDocument.Tags, instance.Document.SelectorTags)
	resp.Diagnostics.Append(d...)
	countries, d := tfconv.StringListFromAPIPreserveNull(ctx, priorDocument.Countries, instance.Document.SelectorCountries)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.Feature = types.StringValue(instance.Feature)
	state.Tags = tags
	state.IsDefaultPolicy = types.BoolValue(instance.IsDefaultPolicy)
	state.Document = &documentModel{
		Detectors:  detectors,
		Categories: categories,
		Regions:    regions,
		Industries: industries,
		Tags:       documentTags,
		Countries:  countries,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *dspmPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update DSPM Policy, unexpected error: missing ID",
		)
		return
	}

	instance, err := r.apiClient.UpdateDSPMPolicy(plan.ID.ValueString(), generatePolicyPayload(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating DSPM Policy",
			"Could not update DSPM Policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.IsDefaultPolicy = types.BoolValue(instance.IsDefaultPolicy)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *dspmPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteDSPMPolicy(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DSPM Policy",
			"Could not delete DSPM Policy, unexpected error: "+err.Error(),
		)
	}
}
