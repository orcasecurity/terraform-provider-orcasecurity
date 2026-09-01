package compliance_framework

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &complianceFrameworksDataSource{}
	_ datasource.DataSourceWithConfigure = &complianceFrameworksDataSource{}
)

type complianceFrameworksDataSource struct {
	apiClient *api_client.APIClient
}

type frameworksDataSourceModel struct {
	Custom      types.Bool       `tfsdk:"custom"`
	Active      types.Bool       `tfsdk:"active"`
	Type        types.String     `tfsdk:"type"`
	DisplayName types.String     `tfsdk:"display_name"`
	Search      types.String     `tfsdk:"search"`
	Frameworks  []frameworkModel `tfsdk:"frameworks"`
}

func NewComplianceFrameworksDataSource() datasource.DataSource {
	return &complianceFrameworksDataSource{}
}

func (d *complianceFrameworksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_frameworks"
}

func (d *complianceFrameworksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (d *complianceFrameworksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists compliance frameworks from GET /api/compliance/frameworks/select. " +
			"Filters are optional and AND-ed client-side (the endpoint takes no query params). " +
			"The `user` selection scope is token-scoped: a plan is stable for a given API token, " +
			"but a different token sees different `user` scopes.",
		Attributes: map[string]schema.Attribute{
			"custom": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, keep only custom (`true`) or built-in (`false`) frameworks.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Description: "If set, keep only frameworks whose `selection_scopes` is non-empty (`true`) or empty (`false`).",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Exact match on framework type (e.g. `Orca Frameworks`). Custom frameworks have a null type and never match.",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Description: "Exact match on display name.",
			},
			"search": schema.StringAttribute{
				Optional:    true,
				Description: "Case-insensitive substring over `display_name` and `description`.",
			},
			"frameworks": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Matching frameworks, sorted by `id`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: frameworkAttributes(false),
				},
			},
		},
	}
}

func frameworkAttributes(idRequired bool) map[string]schema.Attribute {
	id := schema.StringAttribute{
		Computed:    true,
		Description: "Framework id.",
	}
	if idRequired {
		id.Computed = false
		id.Required = true
		id.Description = "Framework id (system id or custom numeric id as a string)."
		id.Validators = []validator.String{stringvalidator.LengthAtLeast(1)}
	}
	return map[string]schema.Attribute{
		"id":           id,
		"display_name": schema.StringAttribute{Computed: true, Description: "Framework display name."},
		"description":  schema.StringAttribute{Computed: true, Description: "Framework description. Null when omitted."},
		"custom":       schema.BoolAttribute{Computed: true, Description: "Whether this is a custom framework."},
		"active":       schema.BoolAttribute{Computed: true, Description: "True iff `selection_scopes` is non-empty."},
		"selection_scopes": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Held scopes (`user`, `organization`), sorted. Empty when the framework is disabled.",
		},
		"type":    schema.StringAttribute{Computed: true, Description: "Framework type. Null for custom frameworks."},
		"version": schema.StringAttribute{Computed: true, Description: "Framework version. Null when omitted."},
		"version_agnostic_display_name": schema.StringAttribute{
			Computed:    true,
			Description: "Display name without version. Null when omitted.",
		},
		"is_ready": schema.BoolAttribute{Computed: true, Description: "Whether the framework is ready. Null when omitted."},
		"framework_cloud_vendors": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Cloud vendors this framework applies to. Null when omitted.",
		},
		"icon_family": schema.StringAttribute{Computed: true, Description: "Icon family. Null when omitted."},
		"orca_end_of_support_date": schema.StringAttribute{
			Computed:    true,
			Description: "End of support date. Null when omitted or unset.",
		},
		"visibility": schema.StringAttribute{Computed: true, Description: "Custom-framework visibility. Null for built-ins."},
	}
}

func (d *complianceFrameworksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config frameworksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := d.apiClient.GetComplianceFrameworkSelections()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read compliance frameworks", err.Error())
		return
	}

	frameworks, diags := filterAndSort(ctx, all, frameworkFilters{
		custom:      config.Custom,
		active:      config.Active,
		typ:         config.Type,
		displayName: config.DisplayName,
		search:      config.Search,
	})
	resp.Diagnostics.Append(diags...)
	config.Frameworks = frameworks
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
