package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ScmUnitListSpec describes one provider's fleet-list data source over SCM
// units (installations / groups / accounts). All rows expose the shared unit
// attributes plus the provider's Extra identity attributes.
type ScmUnitListSpec[A any] struct {
	// TypeNameSuffix is appended to the provider type name,
	// e.g. "_shift_left_bitbucket_accounts".
	TypeNameSuffix string
	Description    string
	// CollectionKey is the single root attribute holding the rows,
	// e.g. "accounts".
	CollectionKey string
	// Extra lists provider identity attributes beyond the shared ones. Only
	// string and int64 types are used; all are Computed.
	Extra map[string]attr.Type
	// List fetches all rows.
	List           func(*api_client.APIClient) ([]A, error)
	ListErrorTitle string
	// Row maps one API row to its account name, shared fields, and extras.
	Row func(a *A) (accountName string, common api_client.ScmUnitCommonFields, extras map[string]attr.Value)
}

// ListValue converts API rows into the data source's list state value.
// Exported through per-package wrappers for unit tests.
func (s ScmUnitListSpec[A]) ListValue(rows []A) (types.List, diag.Diagnostics) {
	attrTypes := SharedScmListUnitAttrTypes()
	for k, t := range s.Extra {
		attrTypes[k] = t
	}
	elems := make([]map[string]attr.Value, len(rows))
	for i := range rows {
		name, common, extras := s.Row(&rows[i])
		m := SharedScmListUnitValues(name, common)
		for k, v := range extras {
			m[k] = v
		}
		elems[i] = m
	}
	return ObjectListFromValues(attrTypes, elems)
}

// NewScmUnitListDataSource builds the data source for a spec.
func NewScmUnitListDataSource[A any](spec ScmUnitListSpec[A]) datasource.DataSource {
	return &scmUnitListDataSource[A]{spec: spec}
}

type scmUnitListDataSource[A any] struct {
	apiClient *api_client.APIClient
	spec      ScmUnitListSpec[A]
}

var (
	_ datasource.DataSource              = &scmUnitListDataSource[struct{}]{}
	_ datasource.DataSourceWithConfigure = &scmUnitListDataSource[struct{}]{}
)

func (ds *scmUnitListDataSource[A]) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + ds.spec.TypeNameSuffix
}

func (ds *scmUnitListDataSource[A]) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *scmUnitListDataSource[A]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	nested := SharedScmListUnitAttrs()
	for k, t := range ds.spec.Extra {
		switch t {
		case types.Int64Type:
			nested[k] = dschema.Int64Attribute{Computed: true}
		default:
			nested[k] = dschema.StringAttribute{Computed: true}
		}
	}
	resp.Schema = dschema.Schema{
		Description: ds.spec.Description,
		Attributes: map[string]dschema.Attribute{
			ds.spec.CollectionKey: dschema.ListNestedAttribute{
				Computed: true,
				NestedObject: dschema.NestedAttributeObject{
					Attributes: nested,
				},
			},
		},
	}
}

func (ds *scmUnitListDataSource[A]) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
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
