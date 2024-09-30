package shift_left_project

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
	_ resource.Resource                = &shiftLeftProjectResource{}
	_ resource.ResourceWithConfigure   = &shiftLeftProjectResource{}
	_ resource.ResourceWithImportState = &shiftLeftProjectResource{}
)

type shiftLeftProjectResource struct {
	apiClient *api_client.APIClient
}

type shiftLeftProjectResourceModel struct {
	ID                               types.String `tfsdk:"id"`
	Name                             types.String `tfsdk:"name"`
	Description                      types.String `tfsdk:"description"`
	Key                              types.String `tfsdk:"key"`
	DefaultPolicies                  types.Bool   `tfsdk:"default_policies"`
	SupportCodeComments              types.String `tfsdk:"support_code_comments_via_cli"`
	SupportCveExceptions             types.String `tfsdk:"support_cve_exceptions_via_cli"`
	SupportSecretDetectionSuppresion types.String `tfsdk:"support_secret_detection_suppression_via_cli"`
	GitDefaultBaselineBranch         types.String `tfsdk:"git_default_baseline_branch"`
	PolicyIds                        []string     `tfsdk:"policies_ids"`
	ExceptionIds                     []string     `tfsdk:"exceptions_ids"`
}

func NewShiftLeftProjectResource() resource.Resource {
	return &shiftLeftProjectResource{}
}

func (r *shiftLeftProjectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_project"
}

func (r *shiftLeftProjectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *shiftLeftProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *shiftLeftProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Shift Left project resource. Projects allow you to organize code risk findings within Orca. You can create projects for certain repos, apps, or environments. You can learn more [here](https://docs.orcasecurity.io/docs/shift-left-security-projects).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Shift Left project ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Shift Left project name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Shift Left project description.",
				Required:    true,
			},
			"key": schema.StringAttribute{
				Description: "Shift Left key.",
				Required:    true,
			},
			"default_policies": schema.BoolAttribute{
				Description: "Whether or not the Orca built-in policies are attached to the project.",
				Required:    true,
			},
			"support_code_comments_via_cli": schema.StringAttribute{
				Description: "Controls whether IaC code comments (for suppressing findings) should be allowed, ignored, or blocked. You can read more about it [here](https://docs.orcasecurity.io/docs/managing-iac-exceptions). Possible values are BLOCK, ALLOW, and IGNORE.",
				Optional:    true,
			},
			"support_cve_exceptions_via_cli": schema.StringAttribute{
				Description: "Control whether CVEs exception management via code should be allowed or blocked. Possible values are BLOCK and ALLOW. ALLOW: an exception file can be passed to the CLI execution in order to suppress issues. BLOCK: the scan will fail when exceptions are defined and specified in the CLI execution.",
				Optional:    true,
			},
			"support_secret_detection_suppression_via_cli": schema.StringAttribute{
				Description: "Control whether code comments or exception handling via config file to suppress found secrets should be allowed, ignored, or blocked. Possible values are BLOCK, ALLOW, and IGNORE. If BLOCK is specified, the scan will fail if issues are found that are ignored via code comments or the exception configuration file.",
				Optional:    true,
			},
			"git_default_baseline_branch": schema.StringAttribute{
				Description: "By default, the main or master branch is used to capture the baseline. If you need to select a different branch that will serve as your project's/repository's main (protect) branch, specify it here. You can read more [here](https://docs.orcasecurity.io/docs/shift-left-baseline).",
				Optional:    true,
			},
			"policies_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Policies to attach to this project, specified by their IDs.",
				Optional:    true,
			},
			"exceptions_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Description: "Exception lists to attach to this project, specified by their IDs.",
				Optional:    true,
			},
		},
	}
}

func (r *shiftLeftProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shiftLeftProjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := api_client.ShiftLeftProject{
		Name:                             plan.Name.ValueString(),
		Description:                      plan.Description.ValueString(),
		Key:                              plan.Key.ValueString(),
		DefaultPolicies:                  plan.DefaultPolicies.ValueBool(),
		SupportCodeComments:              plan.SupportCodeComments.ValueString(),
		SupportCveExceptions:             plan.SupportCveExceptions.ValueString(),
		SupportSecretDetectionSuppresion: plan.SupportSecretDetectionSuppresion.ValueString(),
		GitDefaultBaselineBranch:         plan.GitDefaultBaselineBranch.ValueString(),
		PolicyIds:                        plan.PolicyIds,
		ExceptionIds:                     plan.ExceptionIds,
	}

	instance, err := r.apiClient.CreateShiftLeftProject(createReq)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Shift Left project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(instance.ID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shiftLeftProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state shiftLeftProjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesShiftLeftProjectExist(state.ID.ValueString())
	tflog.Error(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Shift Left project",
			fmt.Sprintf("Could not read Shift Left project ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Project %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetShiftLeftProject(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	tflog.Error(ctx, instance.ID)
	state.Name = types.StringValue(instance.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shiftLeftProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan shiftLeftProjectResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Shift Left project, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	updateReq := api_client.ShiftLeftProject{
		ID:                               plan.ID.ValueString(),
		Name:                             plan.Name.ValueString(),
		Description:                      plan.Description.ValueString(),
		Key:                              plan.Key.ValueString(),
		DefaultPolicies:                  plan.DefaultPolicies.ValueBool(),
		SupportCodeComments:              plan.SupportCodeComments.ValueString(),
		SupportCveExceptions:             plan.SupportCveExceptions.ValueString(),
		SupportSecretDetectionSuppresion: plan.SupportSecretDetectionSuppresion.ValueString(),
		GitDefaultBaselineBranch:         plan.GitDefaultBaselineBranch.ValueString(),
		PolicyIds:                        plan.PolicyIds,
		ExceptionIds:                     plan.ExceptionIds,
	}

	_, err := r.apiClient.UpdateShiftLeftProject(updateReq.ID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	_, err = r.apiClient.GetShiftLeftProject(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project",
			"Could not read project ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shiftLeftProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shiftLeftProjectResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteShiftLeftProject(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Shift Left project",
			"Could not delete Shift Left project, unexpected error: "+err.Error(),
		)
		return
	}
}
