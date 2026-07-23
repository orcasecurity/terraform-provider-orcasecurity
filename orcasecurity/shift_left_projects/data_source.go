package shift_left_projects

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &projectsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectsDataSource{}
)

type projectsDataSource struct {
	apiClient *api_client.APIClient
}

type projectsModel struct {
	Projects types.List `tfsdk:"projects"`
}

func NewProjectsDataSource() datasource.DataSource { return &projectsDataSource{} }

func (ds *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_projects"
}

func (ds *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

var projectAttrTypes = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
	"key":  types.StringType,
}

func (ds *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: "Lists every Orca shift-left project in the organization, for fleet-wide for_each (e.g. attaching a policy to all projects).",
		Attributes: map[string]dschema.Attribute{
			"projects": dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: map[string]dschema.Attribute{
						"id":   dschema.StringAttribute{Computed: true},
						"name": dschema.StringAttribute{Computed: true},
						"key":  dschema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func projectsToListValue(projects []api_client.ShiftLeftProjectSummary) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: projectAttrTypes}
	elems := make([]attr.Value, len(projects))
	for i, p := range projects {
		obj, d := types.ObjectValue(projectAttrTypes, map[string]attr.Value{
			"id":   types.StringValue(p.ID),
			"name": types.StringValue(p.Name),
			"key":  types.StringValue(p.Key),
		})
		diags.Append(d...)
		elems[i] = obj
	}
	list, d := types.ListValue(elemType, elems)
	diags.Append(d...)
	return list, diags
}

func (ds *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	projects, err := ds.apiClient.ListShiftLeftProjects()
	if err != nil {
		resp.Diagnostics.AddError("Error listing shift-left projects", err.Error())
		return
	}
	list, diags := projectsToListValue(projects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &projectsModel{Projects: list})...)
}
