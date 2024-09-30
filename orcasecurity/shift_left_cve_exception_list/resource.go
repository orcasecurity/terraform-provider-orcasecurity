package shift_left_cve_exception_list

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

/*
 * ResourceWithConfigure is an interface type that extends Resource to include a method which the framework
 * will automatically call so provider developers have the opportunity to setup any necessary provider-level data or clients in the Resource type.
 *
 * resource.ResourceWithImportState: Optional interface on top of Resource that enables provider control
 * over the ImportResourceState RPC. This RPC is called by Terraform when the `terraform import` command
 * is executed. Afterwards, the ReadResource RPC is executed to allow providers to fully populate the resource state.
 *
 */

var (
	_ resource.Resource                = &shiftLeftCveExceptionListResource{}
	_ resource.ResourceWithConfigure   = &shiftLeftCveExceptionListResource{}
	_ resource.ResourceWithImportState = &shiftLeftCveExceptionListResource{}
)

type shiftLeftCveExceptionListResource struct {
	apiClient *api_client.APIClient
}

type Vulnerability struct {
	CVEID          types.String   `tfsdk:"cve_id"`
	Description    types.String   `tfsdk:"description"`
	Expiration     types.String   `tfsdk:"expiration"`
	Disabled       types.Bool     `tfsdk:"disabled"`
	RepositoryURLs []types.String `tfsdk:"repositories_urls"`
}

type Project struct {
	ProjectID   types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"name"`
	ProjectKey  types.String `tfsdk:"key"`
}

type ShiftLeftCveExceptionList struct {
	ID              types.String    `tfsdk:"id"`
	Name            types.String    `tfsdk:"name"`
	Description     types.String    `tfsdk:"description"`
	Disabled        types.Bool      `tfsdk:"disabled"`
	Vulnerabilities []Vulnerability `tfsdk:"vulnerabilities"`
	Projects        []Project       `tfsdk:"projects"`
}

func NewShiftLeftCveExceptionListResource() resource.Resource {
	return &shiftLeftCveExceptionListResource{}
}

func (r *shiftLeftCveExceptionListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_cve_exception_list"
}

func (r *shiftLeftCveExceptionListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *shiftLeftCveExceptionListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *shiftLeftCveExceptionListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a Shift Left CVE exception list resource. CVE Exception Lists allow you to ignore certain vulnerabilities that have been identified in scans of your Shift Left Projects. You can read more [here](https://docs.orcasecurity.io/docs/shift-left-security-managing-exceptions).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Shift Left exception list ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Shift Left exception list name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Shift Left exception list description.",
				Optional:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Whether or not the exception list is disabled.",
				Required:    true,
			},
			"vulnerabilities": schema.ListNestedAttribute{
				Description: "Vulnerabilities that compose this exception list.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cve_id": schema.StringAttribute{
							Required:    true,
							Description: "CVE ID of the vulnerability to except.",
						},
						"description": schema.StringAttribute{
							Required:    true,
							Description: "Description (or justification, if that's how you want to use this field) of this exception within the exception list.",
						},
						"expiration": schema.StringAttribute{
							Optional:    true,
							Description: "Expiration date. Format should be \"YYYY/MM/DD\". To permanently exclude the vulnerability, do not use this field. To temporarily exclude the vulnerability, specify an Expiration Date. After this date, the vulnerability is no longer excluded.",
						},
						"disabled": schema.BoolAttribute{
							Required:    true,
							Description: "Whether or not this vulnerability within the exception list is disabled.",
						},
						"repositories_urls": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
							Description: "[NOT YET SUPPORTED] Code repositories (identified by their URLs) to associate with this exception list.",
						},
					},
				},
			},
			"projects": schema.ListNestedAttribute{
				Description: "[Not yet supported] Projects to which this exception list applies.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The project's ID.",
						},
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The project's name.",
						},
						"key": schema.StringAttribute{
							Required:    true,
							Description: "The project's key.",
						},
					},
				},
			},
		},
	}
}

func (r *shiftLeftCveExceptionListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ShiftLeftCveExceptionList
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	//var finalDiags diag.Diagnostics

	var projects []api_client.Project
	for _, item := range plan.Projects {
		projects = append(projects, api_client.Project{
			ProjectID:   item.ProjectID.ValueString(),
			ProjectName: item.ProjectName.ValueString(),
			ProjectKey:  item.ProjectKey.ValueString(),
		})
	}

	var vulnerabilities []api_client.Vulnerability
	for _, item := range plan.Vulnerabilities {
		/*var repository_urls []string
		if !item.RepositoryURLs.IsNull() {
			diags := item.RepositoryURLs.ElementsAs(ctx, &repository_urls, false)
			finalDiags.Append(diags...)
		}*/
		if item.Expiration.ValueString() == "" {
			vulnerabilities = append(vulnerabilities, api_client.Vulnerability{
				CVEID:       item.CVEID.ValueString(),
				Description: item.Description.ValueString(),
				Disabled:    item.Disabled.ValueBool(),
				Expiration:  item.Expiration.ValueString(),
				//RepositoryURLs: item.RepositoryURLs,
			})
		} else {
			vulnerabilities = append(vulnerabilities, api_client.Vulnerability{
				CVEID:       item.CVEID.ValueString(),
				Description: item.Description.ValueString(),
				Disabled:    item.Disabled.ValueBool(),
				Expiration:  item.Expiration.ValueString(),
				//RepositoryURLs: item.RepositoryURLs,
			})
		}

	}

	exceptionListToCreate := api_client.ShiftLeftCveExceptionList{
		Name:            plan.Name.ValueString(),
		Description:     plan.Description.ValueString(),
		Disabled:        plan.Disabled.ValueBool(),
		Projects:        projects,
		Vulnerabilities: vulnerabilities,
	}

	instance, err := r.apiClient.CreateShiftLeftCveExceptionList(exceptionListToCreate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating exception list",
			"Could not create exception list, unexpected error: "+err.Error(),
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

func (r *shiftLeftCveExceptionListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ShiftLeftCveExceptionList
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.apiClient.DoesShiftLeftCveExceptionListExist(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Shift Left CVE exception list",
			fmt.Sprintf("Could not read group ID %s: %s", state.ID, err.Error()),
		)
		return
	}
	if !exists {
		tflog.Warn(ctx, fmt.Sprintf("Exception list %s is missing on the remote side.", state.ID))
		resp.State.RemoveResource(ctx)
		return
	}

	instance, err := r.apiClient.GetShiftLeftCveExceptionList(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Shift Left CVE exception list",
			"Could not read exception list, unexpected error: "+err.Error(),
		)
		return
	}

	state.ID = types.StringValue(instance.ID)
	state.Name = types.StringValue(instance.Name)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shiftLeftCveExceptionListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ShiftLeftCveExceptionList
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	//var finalDiags diag.Diagnostics

	if plan.ID == types.StringValue("") {
		resp.Diagnostics.AddError(
			"Error updating Shift Left CVE exception list",
			"Could not update exception list, unexpected error: ",
		)
		return
	}

	/*var projects []api_client.Project
	for _, item := range plan.Projects {
		projects = append(projects, api_client.Project{
			ProjectID:   item.ProjectID.ValueString(),
			ProjectName: item.ProjectName.ValueString(),
			ProjectKey:  item.ProjectKey.ValueString(),
		})
	}*/

	var vulnerabilities []api_client.Vulnerability
	for _, item := range plan.Vulnerabilities {
		//var repository_urls []string
		/*if len(item.RepositoryURLs) > 0 {
			diags := item.RepositoryURLs.ElementsAs(ctx, &repository_urls, false)
			finalDiags.Append(diags...)
		}*/

		vulnerabilities = append(vulnerabilities, api_client.Vulnerability{
			CVEID:       item.CVEID.ValueString(),
			Description: item.Description.ValueString(),
			Disabled:    item.Disabled.ValueBool(),
			Expiration:  item.Expiration.ValueString(),
		})
	}

	updateReq := api_client.ShiftLeftCveExceptionList{
		ID:          plan.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Disabled:    plan.Disabled.ValueBool(),
		//Projects:        projects,
		Vulnerabilities: vulnerabilities,
	}

	_, err := r.apiClient.UpdateShiftLeftCveExceptionList(updateReq.ID, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Shift Left exception list",
			"Could not update Shift Left exception list, unexpected error: "+err.Error(),
		)
		return
	}

	/*_, err = r.apiClient.GetShiftLeftCveExceptionList(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading exception list",
			"Could not read exception list ID: "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}*/

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *shiftLeftCveExceptionListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ShiftLeftCveExceptionList
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.apiClient.DeleteShiftLeftCveExceptionList(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting exception list",
			"Could not delete exception list, unexpected error: "+err.Error(),
		)
		return
	}
}
