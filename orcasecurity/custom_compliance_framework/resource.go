package custom_compliance_framework

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
	_ resource.Resource                = &customComplianceFrameworkResource{}
	_ resource.ResourceWithConfigure   = &customComplianceFrameworkResource{}
	_ resource.ResourceWithImportState = &customComplianceFrameworkResource{}
)

type customComplianceFrameworkResource struct {
	apiClient *api_client.APIClient
}

type testModel struct {
	RuleID            types.String `tfsdk:"rule_id"`
	RuleIDInFramework types.String `tfsdk:"rule_id_in_framework"`
}

type sectionModel struct {
	Name  types.String `tfsdk:"name"`
	Tests []testModel  `tfsdk:"tests"`
}

type customComplianceFrameworkResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Sections    []sectionModel `tfsdk:"sections"`
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

func (r *customComplianceFrameworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a custom compliance framework resource. " +
			"Note: sections and their tests are write-only. The API does not return " +
			"section/test data on read, so drift detection for sections is not possible.",
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
				Required:    true,
				Description: "Framework description.",
			},
			"sections": schema.ListNestedAttribute{
				Required: true,
				Description: "Framework sections containing tests/controls. " +
					"Note: sections are write-only and not returned by the API on read. " +
					"Terraform will preserve the last-applied value in state.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Section name.",
						},
						"tests": schema.ListNestedAttribute{
							Required:    true,
							Description: "Tests (controls) within this section.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"rule_id": schema.StringAttribute{
										Required:    true,
										Description: "The rule ID for the test/control.",
									},
									"rule_id_in_framework": schema.StringAttribute{
										Required:    true,
										Description: "The identifier for this rule within the framework.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *customComplianceFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customComplianceFrameworkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api_client.CustomComplianceFrameworkCreateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Sections:    sectionsToAPI(plan.Sections),
	}

	instance, err := r.apiClient.CreateCustomComplianceFramework(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating custom compliance framework",
			"Could not create custom compliance framework, unexpected error: "+err.Error(),
		)
		return
	}

	id := instance.ID.String()

	// Refresh from GET to get canonical server values
	readInstance, err := r.apiClient.GetCustomComplianceFramework(id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom compliance framework",
			"Could not read custom compliance framework after creation, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(id)
	if readInstance != nil {
		plan.Name = types.StringValue(readInstance.DisplayName)
		plan.Description = types.StringValue(readInstance.Description)
	}
	// Sections are preserved from plan (GET does not return them)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *customComplianceFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customComplianceFrameworkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetCustomComplianceFramework(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom compliance framework",
			fmt.Sprintf("Could not read custom compliance framework ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("Custom compliance framework %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(instance.DisplayName)
	state.Description = types.StringValue(instance.Description)
	// Sections are NOT returned by the API — preserve whatever is in current state

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *customComplianceFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customComplianceFrameworkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := api_client.CustomComplianceFrameworkUpdateRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Sections:    sectionsToAPI(plan.Sections),
	}

	_, err := r.apiClient.UpdateCustomComplianceFramework(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating custom compliance framework",
			"Could not update custom compliance framework, unexpected error: "+err.Error(),
		)
		return
	}

	// Refresh from GET to get canonical server values
	readInstance, err := r.apiClient.GetCustomComplianceFramework(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading custom compliance framework",
			"Could not read custom compliance framework after update, unexpected error: "+err.Error(),
		)
		return
	}

	if readInstance != nil {
		plan.Name = types.StringValue(readInstance.DisplayName)
		plan.Description = types.StringValue(readInstance.Description)
	}
	// Sections are preserved from plan (GET does not return them)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *customComplianceFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customComplianceFrameworkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteCustomComplianceFramework(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting custom compliance framework",
			"Could not delete custom compliance framework, unexpected error: "+err.Error(),
		)
		return
	}
}

// sectionsToAPI converts Terraform section models to API request structs.
func sectionsToAPI(sections []sectionModel) []api_client.CustomComplianceFrameworkSection {
	var result []api_client.CustomComplianceFrameworkSection
	for _, s := range sections {
		var tests []api_client.CustomComplianceFrameworkTest
		for _, t := range s.Tests {
			tests = append(tests, api_client.CustomComplianceFrameworkTest{
				RuleID:            t.RuleID.ValueString(),
				RuleIDInFramework: t.RuleIDInFramework.ValueString(),
			})
		}
		result = append(result, api_client.CustomComplianceFrameworkSection{
			Name:  s.Name.ValueString(),
			Tests: tests,
		})
	}
	return result
}
