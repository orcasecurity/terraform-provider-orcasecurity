package admission_controller

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource                     = &templateDataSource{}
	_ datasource.DataSourceWithConfigure        = &templateDataSource{}
	_ datasource.DataSourceWithConfigValidators = &templateDataSource{}
)

type templateDataSource struct {
	apiClient *api_client.APIClient
}

type templateDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	Source         types.String `tfsdk:"source"`
	ControllerType types.String `tfsdk:"controller_type"`
	Version        types.String `tfsdk:"version"`
	Description    types.String `tfsdk:"description"`
	SupportedKinds types.List   `tfsdk:"supported_kinds"`
}

func NewAdmissionControllerTemplateDataSource() datasource.DataSource {
	return &templateDataSource{}
}

func (ds *templateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *templateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admission_controller_template"
}

func (ds *templateDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("name"),
			path.MatchRoot("display_name"),
		),
	}
}

func (ds *templateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up an Admission Controller control template by name or display name. " +
			"Use its `id` as the `template_id` of an `orcasecurity_admission_controller_control` resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Template ID.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Template internal name (e.g. `k8sallowedrepos`). Exactly one of `name` and `display_name` must be set.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Template display name as shown in the Orca UI (e.g. `Allowed Container Registries`). Exactly one of `name` and `display_name` must be set.",
			},
			"source": schema.StringAttribute{
				Computed:    true,
				Description: "Template source: `internal` or `custom`.",
			},
			"controller_type": schema.StringAttribute{
				Computed:    true,
				Description: "Controller type: `gatekeeper` or `kyverno`.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "Template version.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Template description.",
			},
			"supported_kinds": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Kubernetes resource kinds this template supports (valid values for the control's `cluster_scope.kinds[].kinds`).",
			},
		},
	}
}

func (ds *templateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state templateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	templates, err := ds.apiClient.GetAdmissionControllerTemplates()
	if err != nil {
		resp.Diagnostics.AddError("Error reading admission controller templates", err.Error())
		return
	}

	var matches []api_client.AdmissionControllerTemplate
	for _, template := range templates {
		if !state.Name.IsNull() && template.Name == state.Name.ValueString() {
			matches = append(matches, template)
		}
		if !state.DisplayName.IsNull() && template.DisplayName == state.DisplayName.ValueString() {
			matches = append(matches, template)
		}
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError(
			"Admission controller template not found",
			"No template matches the given name/display_name. List templates via GET /api/admission_controller/templates.",
		)
		return
	}
	if len(matches) > 1 {
		resp.Diagnostics.AddError(
			"Ambiguous admission controller template",
			fmt.Sprintf("%d templates match; use `name` (unique internal name) instead of `display_name`.", len(matches)),
		)
		return
	}

	template := matches[0]
	state.ID = types.StringValue(template.ID)
	state.Name = types.StringValue(template.Name)
	state.DisplayName = types.StringValue(template.DisplayName)
	state.Source = types.StringValue(template.Source)
	state.ControllerType = types.StringValue(template.ControllerType)
	state.Version = types.StringValue(template.Version)
	state.Description = types.StringValue(template.Description)

	supportedKinds, diags := types.ListValueFrom(ctx, types.StringType, template.SupportedKinds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.SupportedKinds = supportedKinds

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
