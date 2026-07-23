package shift_left_integration

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SharedScmListUnitAttrs are the nested attributes common to every SCM list
// data source element (plus provider-specific identity keys callers add).
func SharedScmListUnitAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"account_name":      schema.StringAttribute{Computed: true},
		"installation_mode": schema.StringAttribute{Computed: true},
		"default_policies":  schema.BoolAttribute{Computed: true},
		"integration_status": schema.StringAttribute{
			Computed:    true,
			Description: "Live integration health from the API when present.",
		},
	}
}

// SharedScmListUnitAttrTypes matches SharedScmListUnitAttrs for ObjectValue builds.
func SharedScmListUnitAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"account_name":       types.StringType,
		"installation_mode":  types.StringType,
		"default_policies":   types.BoolType,
		"integration_status": types.StringType,
	}
}

// SharedScmListUnitValues fills the shared keys for a list element.
func SharedScmListUnitValues(accountName, mode, integrationStatus string, defaultPolicies bool) map[string]attr.Value {
	return map[string]attr.Value{
		"account_name":       types.StringValue(accountName),
		"installation_mode":  types.StringValue(mode),
		"default_policies":   types.BoolValue(defaultPolicies),
		"integration_status": OptionalID(integrationStatus),
	}
}

// ObjectListFromValues builds a typed list of objects from per-element attr maps.
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
