package shift_left_integration

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"

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
		"scan_all_state": schema.StringAttribute{
			Computed:    true,
			Description: "State of the scan-all onboarding flow when present.",
		},
		"integrated_repositories_count": schema.Int64Attribute{
			Computed:    true,
			Description: "Count of repositories integrated under this unit.",
		},
		"scm_posture_policy_id": schema.StringAttribute{
			Computed:    true,
			Description: "ID of the SCM posture policy attached to this unit when present.",
		},
	}
}

// SharedScmListUnitAttrTypes matches SharedScmListUnitAttrs for ObjectValue builds.
func SharedScmListUnitAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"account_name":                  types.StringType,
		"installation_mode":             types.StringType,
		"default_policies":              types.BoolType,
		"integration_status":            types.StringType,
		"scan_all_state":                types.StringType,
		"integrated_repositories_count": types.Int64Type,
		"scm_posture_policy_id":         types.StringType,
	}
}

// SharedScmListUnitValues fills the shared keys for a list element.
func SharedScmListUnitValues(accountName string, u api_client.ScmUnitCommonFields) map[string]attr.Value {
	return map[string]attr.Value{
		"account_name":                  types.StringValue(accountName),
		"installation_mode":             types.StringValue(u.InstallationMode),
		"default_policies":              types.BoolValue(u.DefaultPolicies),
		"integration_status":            OptionalID(u.IntegrationStatus),
		"scan_all_state":                OptionalID(u.ScanAllState),
		"integrated_repositories_count": types.Int64Value(u.IntegratedRepositoriesCount),
		"scm_posture_policy_id":         OptionalID(u.ScmPosturePolicyID),
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
