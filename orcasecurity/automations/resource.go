package automations

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
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

var (
	_ resource.Resource                     = &automationResource{}
	_ resource.ResourceWithConfigure        = &automationResource{}
	_ resource.ResourceWithImportState      = &automationResource{}
	_ resource.ResourceWithConfigValidators = &automationResource{}
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

type automationJiraIssueModel struct {
	TemplateName  types.String `tfsdk:"template_name"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationAzureDevopsWorkItemModel struct {
	TemplateName  types.String `tfsdk:"template_name"`
	ParentIssueID types.String `tfsdk:"parent_issue"`
}

type automationSumoLogicModel struct {
}

type automationWebhookModel struct {
	Name types.String `tfsdk:"name"`
}

type automationResourceModel struct {
	ID                  types.String                        `tfsdk:"id"`
	Name                types.String                        `tfsdk:"name"`
	Description         types.String                        `tfsdk:"description"`
	Query               *automationQueryModel               `tfsdk:"query"`
	JiraIssue           *automationJiraIssueModel           `tfsdk:"jira_issue"`
	AzureDevopsWorkItem *automationAzureDevopsWorkItemModel `tfsdk:"azure_devops_work_item"`
	SumoLogic           *automationSumoLogicModel           `tfsdk:"sumologic"`
	Webhook             *automationWebhookModel             `tfsdk:"webhook"`
	OrganizationID      types.String                        `tfsdk:"organization_id"`
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

func (r *automationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("jira_issue"),
			path.MatchRoot("azure_devops_work_item"),
			path.MatchRoot("sumologic"),
			path.MatchRoot("webhook"),
		),
	}
}

func (r *automationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *automationResource) Schema(_ context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Provider Orca Security automation resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "Automation ID.",
			},
			"organization_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Automation name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Automation description.",
				Optional:    true,
			},
			"query": schema.SingleNestedAttribute{
				Description: "The query to fetch the alerts.",
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
			"jira_issue": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Create a Jira ticket using template.",
				Attributes: map[string]schema.Attribute{
					"template_name": schema.StringAttribute{
						Required:    true,
						Description: "A Jira issue template to use.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under parent issue.",
					},
				},
			},
			"azure_devops_work_item": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Create a Azure Devops Work Item using template.",
				Attributes: map[string]schema.Attribute{
					"template_name": schema.StringAttribute{
						Required:    true,
						Description: "An ADO work item template to use.",
					},
					"parent_issue": schema.StringAttribute{
						Optional:    true,
						Description: "Automatically nest under parent issue.",
					},
				},
			},
			"sumologic": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "SumoLogic integration",
				Attributes:  map[string]schema.Attribute{},
			},
			"webhook": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Notify via Web hook.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Webhook name.",
					},
				},
			},
		},
	}
}

func generateFilterRules(ctx context.Context, plan *automationQueryModel) (api_client.AutomationQuery, diag.Diagnostics) {
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

func generateActions(plan *automationResourceModel) []api_client.AutomationAction {
	var actions []api_client.AutomationAction
	if plan.JiraIssue != nil && !plan.JiraIssue.TemplateName.IsNull() {
		payload := make(map[string]interface{})
		payload["template"] = plan.JiraIssue.TemplateName.ValueString()
		if !plan.JiraIssue.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.JiraIssue.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationJiraActionID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.AzureDevopsWorkItem != nil && !plan.AzureDevopsWorkItem.TemplateName.IsNull() {
		payload := make(map[string]interface{})
		payload["template"] = plan.AzureDevopsWorkItem.TemplateName.ValueString()
		if !plan.AzureDevopsWorkItem.ParentIssueID.IsNull() {
			payload["parent_id"] = plan.AzureDevopsWorkItem.ParentIssueID.ValueString()
		}

		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationAzureDevopsActionID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.SumoLogic != nil {
		payload := make(map[string]interface{})
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationSumoLogicID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
		})
	}

	if plan.Webhook != nil {
		payload := make(map[string]interface{})
		payload["template"] = plan.Webhook.Name.ValueString()
		actions = append(actions, api_client.AutomationAction{
			Type:           api_client.AutomationWebhookID,
			OrganizationID: plan.OrganizationID.ValueString(),
			Data:           payload,
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

	actions := generateActions(&plan)

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

	exists, err := r.apiClient.IsAutomationExists(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Automation",
			fmt.Sprintf("Could not read Automation ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Automation %s is missing on the remote side.", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
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

	// update query filters
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
	if resp.Diagnostics.HasError() {
		return
	}

	state.Query = &automationQueryModel{Filter: filterRules}

	// update actions
	for _, action := range instance.Actions {
		if action.IsJiraIssue() {
			state.JiraIssue = &automationJiraIssueModel{
				TemplateName: types.StringValue(action.Data["template"].(string)),
			}
			if action.Data["parent_id"] != nil {
				state.JiraIssue.ParentIssueID = types.StringValue(action.Data["parent_id"].(string))
			}
		}

		if action.IsAzureDevopsWorkItem() {
			state.AzureDevopsWorkItem = &automationAzureDevopsWorkItemModel{
				TemplateName: types.StringValue(action.Data["template"].(string)),
			}
			if action.Data["parent_id"] != nil {
				state.AzureDevopsWorkItem.ParentIssueID = types.StringValue(action.Data["parent_id"].(string))
			}
		}

		if action.IsSumoLogic() {
			state.SumoLogic = &automationSumoLogicModel{}
		}

		if action.IsWebhook() {
			state.Webhook = &automationWebhookModel{
				Name: types.StringValue(action.Data["template"].(string)),
			}
		}
	}

	diags = resp.State.Set(ctx, &state)
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

	actions := generateActions(&plan)

	updateReq := api_client.Automation{
		Actions:        actions,
		Query:          filterQuery,
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
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
