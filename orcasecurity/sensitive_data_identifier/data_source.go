package sensitive_data_identifier

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &sensitiveDataIdentifiersDataSource{}
	_ datasource.DataSourceWithConfigure = &sensitiveDataIdentifiersDataSource{}
)

type sensitiveDataIdentifiersDataSource struct {
	apiClient *api_client.APIClient
}

type identifiersDataSourceModel struct {
	Title       types.String `tfsdk:"title"`
	Category    types.String `tfsdk:"category"`
	SubCategory types.String `tfsdk:"sub_category"`
	Identifiers types.List   `tfsdk:"identifiers"`
}

func NewSensitiveDataIdentifiersDataSource() datasource.DataSource {
	return &sensitiveDataIdentifiersDataSource{}
}

func (ds *sensitiveDataIdentifiersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *sensitiveDataIdentifiersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sensitive_data_identifiers"
}

func (ds *sensitiveDataIdentifiersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists DSPM sensitive data identifiers (built-in and custom) from GET /api/scan_configuration/dspm_detector. Use it to look up built-in identifier IDs (e.g. `AUS_TAX_NUMBER`) for `orcasecurity_dspm_policy.document.detectors`. Pick an id with a for expression, e.g. one([for i in data.orcasecurity_sensitive_data_identifiers.all.identifiers : i.id if i.title == \"Email Address\"]).",
		Attributes: map[string]schema.Attribute{
			"title": schema.StringAttribute{
				Description: "Filter by identifier title.",
				Optional:    true,
			},
			"category": schema.StringAttribute{
				Description: "Filter by category (`PII`, `PHI`, `PCI`, `SECRET`, `OTHER`).",
				Optional:    true,
			},
			"sub_category": schema.StringAttribute{
				Description: "Filter by sub-category.",
				Optional:    true,
			},
			"identifiers": schema.ListNestedAttribute{
				Description: "Each object is one sensitive data identifier from the API.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Identifier ID (catalog string for built-ins, UUID for custom identifiers).",
						},
						"title": schema.StringAttribute{
							Computed:    true,
							Description: "Identifier title.",
						},
						"category": schema.StringAttribute{
							Computed:    true,
							Description: "Data category.",
						},
						"sub_category": schema.StringAttribute{
							Computed:    true,
							Description: "Data sub-category.",
						},
						"is_custom": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this is a custom (organization-created) identifier.",
						},
						"enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the identifier is enabled.",
						},
					},
				},
			},
		},
	}
}

var identifierAttrTypes = map[string]attr.Type{
	"id":           types.StringType,
	"title":        types.StringType,
	"category":     types.StringType,
	"sub_category": types.StringType,
	"is_custom":    types.BoolType,
	"enabled":      types.BoolType,
}

func identifiersToListValue(detectors []api_client.DSPMDetector) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: identifierAttrTypes}

	elems := make([]attr.Value, len(detectors))
	for i, detector := range detectors {
		obj, d := types.ObjectValue(identifierAttrTypes, map[string]attr.Value{
			"id":           types.StringValue(detector.ID),
			"title":        types.StringValue(detector.Title),
			"category":     types.StringValue(detector.Category),
			"sub_category": types.StringValue(detector.SubCategory),
			"is_custom":    types.BoolValue(detector.IsCustom),
			"enabled":      types.BoolValue(!detector.IsDisabled),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	if diags.HasError() {
		return types.ListNull(elemType), diags
	}
	listVal, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return listVal, diags
}

func (ds *sensitiveDataIdentifiersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config identifiersDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	detectors, err := ds.apiClient.ListDSPMDetectors(api_client.DSPMDetectorListFilters{
		Title:       config.Title.ValueString(),
		Category:    config.Category.ValueString(),
		SubCategory: config.SubCategory.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to read sensitive data identifiers", err.Error())
		return
	}

	listVal, d := identifiersToListValue(detectors)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.Identifiers = listVal
	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
