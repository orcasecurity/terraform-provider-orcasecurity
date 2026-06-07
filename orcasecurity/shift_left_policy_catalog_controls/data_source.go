package shift_left_policy_catalog_controls

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
	_ datasource.DataSource              = &catalogControlsDataSource{}
	_ datasource.DataSourceWithConfigure = &catalogControlsDataSource{}
)

type catalogControlsDataSource struct {
	apiClient *api_client.APIClient
}

type catalogControlsModel struct {
	Type     types.String `tfsdk:"type"`
	Controls types.List   `tfsdk:"controls"`
}

func NewCatalogControlsDataSource() datasource.DataSource {
	return &catalogControlsDataSource{}
}

func (ds *catalogControlsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *catalogControlsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_policy_catalog_controls"
}

func (ds *catalogControlsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists available AppSec policy controls from GET /api/shiftleft/{type}/catalog/controls.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Policy type to fetch controls for (e.g. iac, container_image, sast).",
			},
			"controls": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"title": schema.StringAttribute{
							Computed: true,
						},
						"category": schema.StringAttribute{
							Computed: true,
						},
						"priority": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func controlsToListValue(controls []api_client.CatalogControlSummary) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrTypes := map[string]attr.Type{
		"id":       types.StringType,
		"title":    types.StringType,
		"category": types.StringType,
		"priority": types.StringType,
	}
	elemType := types.ObjectType{AttrTypes: attrTypes}

	elems := make([]attr.Value, len(controls))
	for i, c := range controls {
		obj, d := types.ObjectValue(attrTypes, map[string]attr.Value{
			"id":       types.StringValue(c.ID),
			"title":    types.StringValue(c.Title),
			"category": types.StringValue(c.Category),
			"priority": types.StringValue(c.Priority),
		})
		diags.Append(d...)
		elems[i] = obj
	}

	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

func (ds *catalogControlsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state catalogControlsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	catalog, err := ds.apiClient.GetShiftLeftPolicyCatalogControls(state.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading catalog controls", err.Error())
		return
	}

	controls, diags := controlsToListValue(api_client.FlattenCatalogControls(catalog.Body))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Controls = controls
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
