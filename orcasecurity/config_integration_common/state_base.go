package config_integration_common

import "github.com/hashicorp/terraform-plugin-framework/types"

// CommonFields is meant to be embedded in a per-variant state struct so the cross-variant
// GetCommon/SetCommon implementations live in this package exactly once. Use this variant for
// integrations whose Orca service config does NOT accept “business_units“.
//
//	type state struct {
//	    cc.CommonFields
//	    APIToken types.String `tfsdk:"api_token"`
//	}
type CommonFields struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
}

func (c *CommonFields) GetCommon() *Common {
	return &Common{
		ID:            c.ID,
		TemplateName:  c.TemplateName,
		IsEnabled:     c.IsEnabled,
		IsDefault:     c.IsDefault,
		BusinessUnits: types.SetNull(types.StringType),
	}
}

func (c *CommonFields) SetCommon(v Common) {
	c.ID, c.TemplateName, c.IsEnabled, c.IsDefault = v.ID, v.TemplateName, v.IsEnabled, v.IsDefault
}

// CommonFieldsWithBU is the variant for integrations whose Orca service config accepts the
// “business_units“ field. Embed it the same way you would embed CommonFields.
type CommonFieldsWithBU struct {
	ID            types.String `tfsdk:"id"`
	TemplateName  types.String `tfsdk:"template_name"`
	IsEnabled     types.Bool   `tfsdk:"is_enabled"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	BusinessUnits types.Set    `tfsdk:"business_units"`
}

func (c *CommonFieldsWithBU) GetCommon() *Common {
	return &Common{
		ID:            c.ID,
		TemplateName:  c.TemplateName,
		IsEnabled:     c.IsEnabled,
		IsDefault:     c.IsDefault,
		BusinessUnits: c.BusinessUnits,
	}
}

func (c *CommonFieldsWithBU) SetCommon(v Common) {
	c.ID, c.TemplateName, c.IsEnabled, c.IsDefault, c.BusinessUnits = v.ID, v.TemplateName, v.IsEnabled, v.IsDefault, v.BusinessUnits
}
