package sensitive_data_identifier

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &sensitiveDataIdentifierResource{}
	_ resource.ResourceWithConfigure   = &sensitiveDataIdentifierResource{}
	_ resource.ResourceWithImportState = &sensitiveDataIdentifierResource{}
)

type sensitiveDataIdentifierResource struct {
	apiClient *api_client.APIClient
}

type conditionModel struct {
	Source   types.String `tfsdk:"source"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

type propertiesModel struct {
	Conditions      []conditionModel `tfsdk:"conditions"`
	DetectionTypes  types.List       `tfsdk:"detection_types"`
	Sensitivity     types.String     `tfsdk:"sensitivity"`
	Significance    types.String     `tfsdk:"significance"`
	Keywords        types.List       `tfsdk:"keywords"`
	ExcludeKeywords types.List       `tfsdk:"exclude_keywords"`
	StopWildcards   types.List       `tfsdk:"stop_wildcards"`
	TextThreshold   types.Int64      `tfsdk:"text_threshold"`
	DBThreshold     types.Int64      `tfsdk:"db_threshold"`
	OCRThreshold    types.Int64      `tfsdk:"ocr_threshold"`
	AIThreshold     types.Int64      `tfsdk:"ai_threshold"`
}

type stateModel struct {
	ID             types.String     `tfsdk:"id"`
	OrganizationID types.String     `tfsdk:"organization_id"`
	Title          types.String     `tfsdk:"title"`
	Details        types.String     `tfsdk:"details"`
	Category       types.String     `tfsdk:"category"`
	SubCategory    types.String     `tfsdk:"sub_category"`
	Enabled        types.Bool       `tfsdk:"enabled"`
	Properties     *propertiesModel `tfsdk:"properties"`
}

func NewSensitiveDataIdentifierResource() resource.Resource {
	return &sensitiveDataIdentifierResource{}
}

func (r *sensitiveDataIdentifierResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_sensitive_data_identifier"
}

func (r *sensitiveDataIdentifierResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *sensitiveDataIdentifierResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *sensitiveDataIdentifierResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provides a custom DSPM sensitive data identifier (detector). Sensitive data identifiers describe the data patterns DSPM scanning looks for.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Sensitive data identifier ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Orca organization ID.",
			},
			"title": schema.StringAttribute{
				Description: "Identifier title. Must be unique (case-insensitive) within the organization, including against the built-in catalog identifiers.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"details": schema.StringAttribute{
				Description: "Identifier description.",
				Required:    true,
			},
			"category": schema.StringAttribute{
				Description: "Data category. Valid values are `PII`, `PHI`, `PCI`, `SECRET`, and `OTHER`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PII", "PHI", "PCI", "SECRET", "OTHER"),
				},
			},
			"sub_category": schema.StringAttribute{
				Description: "Data sub-category. Must be one of the Orca catalog sub-categories (e.g. `Phone Number`, `Email Address`, `Patient ID`, `API Keys and Tokens`, `Other`); the server rejects values outside the catalog.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the identifier is enabled. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"properties": schema.SingleNestedAttribute{
				Description: "Detection configuration.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"conditions": schema.ListNestedAttribute{
						Description: "Content-matching conditions. At least one is required.",
						Required:    true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"source": schema.StringAttribute{
									Description: "Condition source. Defaults to `content`.",
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("content"),
								},
								"operator": schema.StringAttribute{
									Description: "Condition operator. Defaults to `match`.",
									Optional:    true,
									Computed:    true,
									Default:     stringdefault.StaticString("match"),
								},
								"value": schema.StringAttribute{
									Description: "Regular expression to match. The server requires the pattern to start and end with a boundary (`\\b`, `^`, `$`, or a non-capturing group) and to contain a named capturing group called `secret`, e.g. `\\b(?P<secret>[0-9]{9})\\b`.",
									Required:    true,
								},
							},
						},
					},
					"detection_types": schema.ListAttribute{
						Description: "Detection engines to run. Valid values are `text`, `db`, `ocr`, and `ai`. Defaults to the server default (`[\"text\", \"db\"]`).",
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.ValueStringsAre(stringvalidator.OneOf("text", "db", "ocr", "ai")),
						},
					},
					"sensitivity": schema.StringAttribute{
						Description: "Data sensitivity. Valid values are `critical`, `high`, `medium`, and `low`.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("critical", "high", "medium", "low"),
						},
					},
					"significance": schema.StringAttribute{
						Description: "Finding significance. Valid values are `Minor`, `Moderate`, and `Major` (capitalized — the server rejects lowercase).",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("Minor", "Moderate", "Major"),
						},
					},
					"keywords": schema.ListAttribute{
						Description: "Keywords that boost detection confidence.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"exclude_keywords": schema.ListAttribute{
						Description: "Keywords that suppress detection.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"stop_wildcards": schema.ListAttribute{
						Description: "Path wildcards excluded from scanning.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"text_threshold": schema.Int64Attribute{
						Description: "Minimum matches for text detection.",
						Optional:    true,
					},
					"db_threshold": schema.Int64Attribute{
						Description: "Minimum matches for database detection.",
						Optional:    true,
					},
					"ocr_threshold": schema.Int64Attribute{
						Description: "Minimum matches for OCR detection.",
						Optional:    true,
					},
					"ai_threshold": schema.Int64Attribute{
						Description: "Minimum matches for AI detection.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func generateDetectorPayload(ctx context.Context, plan stateModel) api_client.DSPMDetector {
	var conditions []api_client.DSPMDetectorCondition
	for _, condition := range plan.Properties.Conditions {
		conditions = append(conditions, api_client.DSPMDetectorCondition{
			Source:   condition.Source.ValueString(),
			Operator: condition.Operator.ValueString(),
			Value:    condition.Value.ValueString(),
		})
	}

	return api_client.DSPMDetector{
		Title:       plan.Title.ValueString(),
		Details:     plan.Details.ValueString(),
		Category:    plan.Category.ValueString(),
		SubCategory: plan.SubCategory.ValueString(),
		IsDisabled:  !plan.Enabled.ValueBool(),
		IsCustom:    true,
		Properties: api_client.DSPMDetectorProperties{
			Conditions:      conditions,
			DetectionTypes:  tfconv.StringListToAPI(ctx, plan.Properties.DetectionTypes),
			Sensitivity:     plan.Properties.Sensitivity.ValueString(),
			Significance:    plan.Properties.Significance.ValueString(),
			Keywords:        tfconv.StringListToAPI(ctx, plan.Properties.Keywords),
			ExcludeKeywords: tfconv.StringListToAPI(ctx, plan.Properties.ExcludeKeywords),
			StopWildcards:   tfconv.StringListToAPI(ctx, plan.Properties.StopWildcards),
			TextThreshold:   tfconv.Int64ToAPIPtr(plan.Properties.TextThreshold),
			DBThreshold:     tfconv.Int64ToAPIPtr(plan.Properties.DBThreshold),
			OCRThreshold:    tfconv.Int64ToAPIPtr(plan.Properties.OCRThreshold),
			AIThreshold:     tfconv.Int64ToAPIPtr(plan.Properties.AIThreshold),
		},
	}
}

// applyComputedFromInstance sets the computed attributes on the plan after
// create/update. detection_types is only overwritten when the planned value
// is unknown (user did not configure it) so a configured value is never
// contradicted after apply.
func applyComputedFromInstance(ctx context.Context, plan *stateModel, instance *api_client.DSPMDetector, diags *diag.Diagnostics) {
	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)
	plan.Enabled = types.BoolValue(!instance.IsDisabled)

	if plan.Properties.DetectionTypes.IsUnknown() || plan.Properties.DetectionTypes.IsNull() {
		detectionTypes, d := types.ListValueFrom(ctx, types.StringType, instance.Properties.DetectionTypes)
		diags.Append(d...)
		plan.Properties.DetectionTypes = detectionTypes
	}
}

func (r *sensitiveDataIdentifierResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.CreateDSPMDetector(generateDetectorPayload(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Sensitive Data Identifier",
			"Could not create Sensitive Data Identifier, unexpected error: "+err.Error(),
		)
		return
	}

	applyComputedFromInstance(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *sensitiveDataIdentifierResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetDSPMDetector(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Sensitive Data Identifier",
			fmt.Sprintf("Could not read Sensitive Data Identifier ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Sensitive Data Identifier %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	prior := propertiesModel{}
	if state.Properties != nil {
		prior = *state.Properties
	}

	var conditions []conditionModel
	for _, condition := range instance.Properties.Conditions {
		conditions = append(conditions, conditionModel{
			Source:   types.StringValue(condition.Source),
			Operator: types.StringValue(condition.Operator),
			Value:    types.StringValue(condition.Value),
		})
	}

	detectionTypes, d := types.ListValueFrom(ctx, types.StringType, instance.Properties.DetectionTypes)
	resp.Diagnostics.Append(d...)
	keywords, d := tfconv.StringListFromAPIPreserveNull(ctx, prior.Keywords, instance.Properties.Keywords)
	resp.Diagnostics.Append(d...)
	excludeKeywords, d := tfconv.StringListFromAPIPreserveNull(ctx, prior.ExcludeKeywords, instance.Properties.ExcludeKeywords)
	resp.Diagnostics.Append(d...)
	stopWildcards, d := tfconv.StringListFromAPIPreserveNull(ctx, prior.StopWildcards, instance.Properties.StopWildcards)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.OrganizationID = types.StringValue(instance.OrganizationID)
	state.Title = types.StringValue(instance.Title)
	state.Details = types.StringValue(instance.Details)
	state.Category = types.StringValue(instance.Category)
	state.SubCategory = types.StringValue(instance.SubCategory)
	state.Enabled = types.BoolValue(!instance.IsDisabled)
	state.Properties = &propertiesModel{
		Conditions:      conditions,
		DetectionTypes:  detectionTypes,
		Sensitivity:     tfconv.StringOrNull(instance.Properties.Sensitivity),
		Significance:    tfconv.StringOrNull(instance.Properties.Significance),
		Keywords:        keywords,
		ExcludeKeywords: excludeKeywords,
		StopWildcards:   stopWildcards,
		TextThreshold:   tfconv.Int64FromAPIPtr(instance.Properties.TextThreshold),
		DBThreshold:     tfconv.Int64FromAPIPtr(instance.Properties.DBThreshold),
		OCRThreshold:    tfconv.Int64FromAPIPtr(instance.Properties.OCRThreshold),
		AIThreshold:     tfconv.Int64FromAPIPtr(instance.Properties.AIThreshold),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *sensitiveDataIdentifierResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Sensitive Data Identifier, unexpected error: missing ID",
		)
		return
	}

	instance, err := r.apiClient.UpdateDSPMDetector(plan.ID.ValueString(), generateDetectorPayload(ctx, plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Sensitive Data Identifier",
			"Could not update Sensitive Data Identifier, unexpected error: "+err.Error(),
		)
		return
	}

	applyComputedFromInstance(ctx, &plan, instance, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *sensitiveDataIdentifierResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteDSPMDetector(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Sensitive Data Identifier",
			"Could not delete Sensitive Data Identifier, unexpected error: "+err.Error(),
		)
	}
}
