package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ScmUnitListSpec adapts a unit collection (accounts, groups) onto the shared
// shift_left_common object-list data source: the shared unit attributes are
// merged with Extra and each row is flattened from ScmUnitCommonFields.
type ScmUnitListSpec[A any] struct {
	TypeNameSuffix string
	Description    string
	CollectionKey  string
	Extra          map[string]attr.Type
	List           func(*api_client.APIClient) ([]A, error)
	ListErrorTitle string
	Row            func(a *A) (accountName string, common api_client.ScmUnitCommonFields, extras map[string]attr.Value)
}

func (s ScmUnitListSpec[A]) objectSpec() shift_left_common.ScmObjectListSpec[A] {
	attrs := SharedScmListUnitAttrs()
	attrTypes := SharedScmListUnitAttrTypes()
	for k, t := range s.Extra {
		attrTypes[k] = t
		switch t {
		case types.Int64Type:
			attrs[k] = dschema.Int64Attribute{Computed: true}
		default:
			attrs[k] = dschema.StringAttribute{Computed: true}
		}
	}
	return shift_left_common.ScmObjectListSpec[A]{
		TypeNameSuffix: s.TypeNameSuffix,
		Description:    s.Description,
		CollectionKey:  s.CollectionKey,
		ListErrorTitle: s.ListErrorTitle,
		Attrs:          attrs,
		AttrTypes:      attrTypes,
		List:           s.List,
		Row: func(a *A) map[string]attr.Value {
			name, common, extras := s.Row(a)
			m := SharedScmListUnitValues(name, common)
			for k, v := range extras {
				m[k] = v
			}
			return m
		},
	}
}

func (s ScmUnitListSpec[A]) ListValue(rows []A) (types.List, diag.Diagnostics) {
	return s.objectSpec().ListValue(rows)
}

func NewScmUnitListDataSource[A any](spec ScmUnitListSpec[A]) datasource.DataSource {
	return shift_left_common.NewScmObjectListDataSource(spec.objectSpec())
}
