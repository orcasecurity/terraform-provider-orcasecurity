package user_preferences

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &userPreferencesDataSource{}
	_ datasource.DataSourceWithConfigure = &userPreferencesDataSource{}
)

type userPreferencesStateModel struct {
	CustomWidgetIDs types.List `tfsdk:"custom_widget_ids"`
	CustomWidgets   types.List `tfsdk:"custom_widgets"`
}

type userPreferencesDataSource struct {
	apiClient *api_client.APIClient
}

func NewUserPreferencesDataSource() datasource.DataSource {
	return &userPreferencesDataSource{}
}

func (ds *userPreferencesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *userPreferencesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_preferences"
}

func (ds *userPreferencesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch user preferences from Orca, including custom widget IDs. Use this to discover existing custom widget IDs for import or reference.",
		Attributes: map[string]schema.Attribute{
			"custom_widget_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "List of custom widget IDs (preference IDs with view_type 'customs_widgets').",
			},
			"custom_widgets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Custom widget ID (preference ID).",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Custom widget name.",
						},
					},
				},
				Description: "List of custom widgets with id and name.",
			},
		},
	}
}

func (ds *userPreferencesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	widgets, err := ds.apiClient.ListCustomWidgets()
	if err != nil {
		resp.Diagnostics.AddError("Unable to list custom widgets", err.Error())
		return
	}

	ids := make([]attr.Value, 0, len(widgets))
	objectElemAttrTypes := map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
	objectElemType := types.ObjectType{}.WithAttributeTypes(objectElemAttrTypes)
	objectElems := make([]attr.Value, 0, len(widgets))

	for _, w := range widgets {
		ids = append(ids, types.StringValue(w.ID))
		obj, diags := types.ObjectValue(objectElemAttrTypes, map[string]attr.Value{
			"id":   types.StringValue(w.ID),
			"name": types.StringValue(w.Name),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		objectElems = append(objectElems, obj)
	}

	idsList, diags := types.ListValue(types.StringType, ids)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	itemsList, diags := types.ListValue(objectElemType, objectElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := userPreferencesStateModel{
		CustomWidgetIDs: idsList,
		CustomWidgets:   itemsList,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
