package compliance_framework

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &complianceFrameworkDataSource{}
	_ datasource.DataSourceWithConfigure = &complianceFrameworkDataSource{}
)

type complianceFrameworkDataSource struct {
	apiClient *api_client.APIClient
}

type singleDataSourceModel struct {
	frameworkModel
	Sections types.List `tfsdk:"sections"`
}

func NewComplianceFrameworkDataSource() datasource.DataSource {
	return &complianceFrameworkDataSource{}
}

func (d *complianceFrameworkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compliance_framework"
}

func (d *complianceFrameworkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (d *complianceFrameworkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := frameworkAttributes(true)
	attrs["sections"] = schema.ListNestedAttribute{
		Computed:    true,
		Description: "Section/test tree from GET /api/compliance/catalog/{id}. Nested at most three levels (a section has tests, or sub-sections, never both). Server-assigned section ids are omitted.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: catalogSectionAttributes(maxCatalogDepth - 1),
		},
	}
	resp.Schema = schema.Schema{
		Description: "Looks up one compliance framework by id, including its catalog section tree. " +
			"Use this to inspect controls without owning the framework.",
		Attributes: attrs,
	}
}

func catalogSectionAttributes(remaining int) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"name": schema.StringAttribute{Computed: true, Description: "Section name."},
		"tests": schema.ListNestedAttribute{
			Computed:    true,
			Description: "Controls in this section. Null when the section only has sub-sections.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"name":                schema.StringAttribute{Computed: true, Description: "Control name. Null when omitted."},
					"rule_id":             schema.StringAttribute{Computed: true, Description: "Sonar rule id."},
					"reference_id":        schema.StringAttribute{Computed: true, Description: "Control id within the framework (`rule_id_in_framework` on write)."},
					"origin_framework_id": schema.StringAttribute{Computed: true, Description: "Origin framework id. Null when omitted."},
					"cloud_vendors": schema.ListAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "Cloud vendors this control applies to. Null when omitted.",
					},
					"control_unique_id": schema.StringAttribute{Computed: true, Description: "Catalog control unique id. Null when omitted."},
					"priority":          schema.StringAttribute{Computed: true, Description: "Control priority. Null when omitted."},
				},
			},
		},
	}
	if remaining > 0 {
		attrs["sections"] = schema.ListNestedAttribute{
			Computed:    true,
			Description: "Nested sub-sections. Null on a leaf. Depth is capped at three levels.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: catalogSectionAttributes(remaining - 1),
			},
		}
	}
	return attrs
}

func (d *complianceFrameworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config singleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.ID.ValueString()
	fw, err := d.apiClient.GetComplianceFramework(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading compliance framework", fmt.Sprintf("Could not read framework %s: %s", id, err.Error()))
		return
	}
	if fw == nil {
		resp.Diagnostics.AddError("Compliance framework not found", fmt.Sprintf("No compliance framework exists for id %q.", id))
		return
	}

	fwModel, diags := frameworkToModel(ctx, *fw)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := singleDataSourceModel{frameworkModel: fwModel}
	catalog, err := d.apiClient.GetComplianceCatalogFramework(id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading compliance catalog", fmt.Sprintf("Could not read catalog for framework %s: %s", id, err.Error()))
		return
	}
	if catalog == nil {
		resp.Diagnostics.AddError("Error reading compliance catalog", fmt.Sprintf("GET /api/compliance/catalog/%s returned no framework.", id))
		return
	}
	sections, diags := catalogSectionsToModel(ctx, catalog.Sections, maxCatalogDepth-1)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Sections = sections
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
