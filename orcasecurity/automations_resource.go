package orcasecurity

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &automationResource{}
	_ resource.ResourceWithConfigure   = &automationResource{}
	_ resource.ResourceWithImportState = &automationResource{}
)

type automationResource struct {
	apiClient *api_client.APIClient
}

type automationQueryRuleModel struct {
	Field    types.String `tfsdk:"field"`
	Includes types.List   `tfsdk:"includes"`
	Excludes types.List   `tfsdk:"excludes"`
}

type automationQueryModel struct {
	Filter []automationQueryRuleModel `tfsdk:"filter"`
}

type automationActionModel struct {
	ID             types.String `tfsdk:"id"`
	Type           types.Int64  `tfsdk:"type"`
	OrganizationID types.String `tfsdk:"organization_id"`
	Data           types.Map    `tfsdk:"data"`
}

type automationJiraIssueModel struct {
	TemplateName  types.String `tfsdk:"template_name"`
	ParentIssueID types.String `tfsdk:"parent_issue_id"`
}

type automationResourceModel struct {
	ID             types.String             `tfsdk:"id"`
	Name           types.String             `tfsdk:"name"`
	Description    types.String             `tfsdk:"description"`
	Query          automationQueryModel     `tfsdk:"query"`
	JiraIssue      automationJiraIssueModel `tfsdk:"jira_issue"`
	OrganizationID types.String             `tfsdk:"organization_id"`
	Actions        []automationActionModel  `tfsdk:"actions"`
}

func NewAutomationResource() resource.Resource {
	return &automationResource{}
}

func (r *automationResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_automation"
}

func (r *automationResource) Configure(_ context.Context, req resource.ConfigureRequest, res *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *automationResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Automation ID",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Automation name",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Automation description",
				Optional:    true,
			},
			"query": schema.SingleNestedAttribute{
				Description: "Trigger query",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"filter": schema.ListNestedAttribute{
						Required: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"field": schema.StringAttribute{
									Required: true,
								},
								"includes": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
								},
								"excludes": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
				},
			},
			"actions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.Int64Attribute{
							Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"organization_id": schema.StringAttribute{
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"data": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"jira_issue": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Create a Jira ticket using template.",
				Attributes: map[string]schema.Attribute{
					"template_name": schema.StringAttribute{
						Required:    true,
						Description: "A Jira issue template to use.",
					},
					"parent_issue_id": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under parent issue.",
					},
				},
			},
		},
	}
}

func generateFilterRules(ctx context.Context, plan automationQueryModel) (api_client.AutomationQuery, diag.Diagnostics) {
	var filterRules []api_client.AutomationFilter
	var finalDiags diag.Diagnostics
	for _, item := range plan.Filter {
		var includes []string
		if !item.Includes.IsNull() {
			diags := item.Includes.ElementsAs(ctx, &includes, false)
			finalDiags.Append(diags...)
		}

		var excludes []string
		if !item.Excludes.IsNull() {
			diags := item.Excludes.ElementsAs(ctx, &excludes, false)
			finalDiags.Append(diags...)
		}

		filterRules = append(filterRules, api_client.AutomationFilter{
			Field:    item.Field.ValueString(),
			Includes: includes,
			Excludes: excludes,
		})
	}
	return api_client.AutomationQuery{Filter: filterRules}, finalDiags
}

func generateActions(plan automationResourceModel) []api_client.AutomationAction {
	var actions []api_client.AutomationAction
	if !plan.JiraIssue.TemplateName.IsNull() {
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationJiraActionID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data: struct {
				Template string `json:"template"`
				ParentID string `json:"parent_id,omitempty"`
			}{
				Template: plan.JiraIssue.TemplateName.ValueString(),
				ParentID: plan.JiraIssue.ParentIssueID.ValueString(),
			},
		})
	}
	return actions
}

func (r *automationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan automationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	filterQuery, filterDiags := generateFilterRules(ctx, plan.Query)
	diags.Append(filterDiags...)

	actions := generateActions(plan)

	createReq := api_client.Automation{
		Actions:     actions,
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Query:       filterQuery,
	}
	instance, err := r.apiClient.CreateAutomation(createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Automation",
			"Could not create Automation, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(instance.ID)
	plan.OrganizationID = types.StringValue(instance.OrganizationID)

	for actionIndex, action := range instance.Actions {
		dataValue, diags := types.MapValueFrom(ctx, types.StringType, action.Data)
		resp.Diagnostics.Append(diags...)

		plan.Actions[actionIndex] = automationActionModel{
			ID:             types.StringValue(action.ID),
			Type:           types.Int64Value(int64(action.Type)),
			OrganizationID: types.StringValue(action.OrganizationID),
			Data:           dataValue,
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state automationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, err := r.apiClient.GetAutomation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			fmt.Sprintf("Could not read Automation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)
	state.Description = types.StringValue(instance.Description)
	state.OrganizationID = types.StringValue(instance.OrganizationID)

	var filterRules []automationQueryRuleModel
	for _, rule := range instance.Query.Filter {
		var includes []types.String
		for _, includeValue := range rule.Includes {
			includes = append(includes, types.StringValue(includeValue))
		}
		includesList, diags := types.ListValueFrom(ctx, types.StringType, includes)
		resp.Diagnostics.Append(diags...)

		var excludes []types.String
		for _, excludeValue := range rule.Excludes {
			excludes = append(excludes, types.StringValue(excludeValue))
		}
		excludesList, diags := types.ListValueFrom(ctx, types.StringType, excludes)
		resp.Diagnostics.Append(diags...)

		filterRules = append(filterRules, automationQueryRuleModel{
			Field:    types.StringValue(rule.Field),
			Includes: includesList,
			Excludes: excludesList,
		})

	}
	state.Query = automationQueryModel{Filter: filterRules}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan automationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"ID is null",
			"Could not update Automation, unexpected error: "+plan.ID.ValueString(),
		)
		return
	}

	filterQuery, filterDiags := generateFilterRules(ctx, plan.Query)
	diags.Append(filterDiags...)

	actions := generateActions(plan)

	tflog.Warn(ctx, fmt.Sprintf("plan id: %s, org id: %s", plan.ID.ValueString(), plan.OrganizationID.ValueString()))

	updateReq := api_client.Automation{
		Actions:        actions,
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		Query:          filterQuery,
		OrganizationID: plan.OrganizationID.ValueString(),
	}

	_, err := r.apiClient.UpdateAutomation(plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Automation",
			"Could not update Automation, unexpected error: "+err.Error(),
		)
		return
	}

	_, err = r.apiClient.GetAutomation(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			"Could not read Automation ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *automationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state automationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteAutomation(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Automation",
			"Could not delete Automation, unexpected error: "+err.Error(),
		)
		return
	}
}
