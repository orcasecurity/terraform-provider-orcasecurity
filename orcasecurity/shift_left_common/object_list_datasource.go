package shift_left_common

import (
	"context"
	"maps"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MergeMaps combines a shared attribute/type/value map with per-provider extras;
// extra wins on conflict.
func MergeMaps[V any](base, extra map[string]V) map[string]V {
	out := make(map[string]V, len(base)+len(extra))
	maps.Copy(out, base)
	maps.Copy(out, extra)
	return out
}

func ObjectListFromValues(attrTypes map[string]attr.Type, elems []map[string]attr.Value) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elemType := types.ObjectType{AttrTypes: attrTypes}
	values := make([]attr.Value, len(elems))
	for i, m := range elems {
		obj, d := types.ObjectValue(attrTypes, m)
		diags.Append(d...)
		values[i] = obj
	}
	list, d := types.ListValue(elemType, values)
	diags.Append(d...)
	return list, diags
}

// ScmObjectListSpec drives the shared read-only list data source used by every
// shift-left collection (installations, repositories, units via an adapter).
type ScmObjectListSpec[A any] struct {
	TypeNameSuffix string
	Description    string
	CollectionKey  string
	ListErrorTitle string
	Attrs          map[string]dschema.Attribute
	AttrTypes      map[string]attr.Type
	List           func(*api_client.APIClient) ([]A, error)
	Row            func(*A) map[string]attr.Value
}

func (s ScmObjectListSpec[A]) ListValue(rows []A) (types.List, diag.Diagnostics) {
	elems := make([]map[string]attr.Value, len(rows))
	for i := range rows {
		elems[i] = s.Row(&rows[i])
	}
	return ObjectListFromValues(s.AttrTypes, elems)
}

func NewScmObjectListDataSource[A any](spec ScmObjectListSpec[A]) datasource.DataSource {
	return &scmObjectListDataSource[A]{spec: spec}
}

type scmObjectListDataSource[A any] struct {
	apiClient *api_client.APIClient
	spec      ScmObjectListSpec[A]
}

var (
	_ datasource.DataSource              = &scmObjectListDataSource[struct{}]{}
	_ datasource.DataSourceWithConfigure = &scmObjectListDataSource[struct{}]{}
)

func (ds *scmObjectListDataSource[A]) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + ds.spec.TypeNameSuffix
}

func (ds *scmObjectListDataSource[A]) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *scmObjectListDataSource[A]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		Description: ds.spec.Description,
		Attributes: map[string]dschema.Attribute{
			ds.spec.CollectionKey: dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: ds.spec.Attrs,
				},
			},
		},
	}
}

func (ds *scmObjectListDataSource[A]) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	rows, err := ds.spec.List(ds.apiClient)
	if err != nil {
		resp.Diagnostics.AddError(ds.spec.ListErrorTitle, err.Error())
		return
	}
	list, diags := ds.spec.ListValue(rows)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ds.spec.CollectionKey), list)...)
}
