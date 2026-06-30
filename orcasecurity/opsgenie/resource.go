package opsgenie

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	ID            types.String `tfsdk:"id"`
	TemplateName  types.String `tfsdk:"template_name"`
	IsEnabled     types.Bool   `tfsdk:"is_enabled"`
	IsDefault     types.Bool   `tfsdk:"is_default"`
	BusinessUnits types.Set    `tfsdk:"business_units"`
	OpsgenieKey   types.String `tfsdk:"opsgenie_key"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: s.BusinessUnits}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault, s.BusinessUnits = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault, c.BusinessUnits
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_opsgenie" }
func (variant) UIName() string              { return "Opsgenie integration" }
func (variant) SupportsBusinessUnits() bool { return true }
func (variant) Description() string {
	return "Manage an Opsgenie integration in Orca. Creates an external service config of `service_name = \"opsgenie\"` and stores the Opsgenie API key in Orca's secret store."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"opsgenie_key": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Opsgenie API integration key. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(ctx context.Context, s *state, diags *diag.Diagnostics) api_client.OpsgenieExternalServiceConfig {
	return api_client.OpsgenieExternalServiceConfig{
		TemplateName:  s.TemplateName.ValueString(),
		IsEnabled:     s.IsEnabled.ValueBool(),
		IsDefault:     s.IsDefault.ValueBool(),
		Config:        api_client.OpsgenieConfig{OpsgenieKey: s.OpsgenieKey.ValueString()},
		BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
	}
}

func apiObj(o *api_client.OpsgenieExternalServiceConfig) cc.APIObject {
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault, BusinessUnits: o.BusinessUnits}
}

func (v variant) Create(c *api_client.APIClient, ctx context.Context, plan cc.State, diags *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	created, err := c.CreateOpsgenieConfig(v.buildPayload(ctx, plan.(*state), diags))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Opsgenie integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	current, err := c.GetOpsgenieConfig(st.GetCommon().TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Opsgenie integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current), true, nil
}

func (v variant) Update(c *api_client.APIClient, ctx context.Context, plan cc.State, templateName string, diags *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	updated, err := c.UpdateOpsgenieConfig(templateName, v.buildPayload(ctx, plan.(*state), diags))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Opsgenie integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteOpsgenieConfig(templateName)
}

type opsgenieResource struct{ cc.Resource }

func NewOpsgenieResource() resource.Resource {
	return &opsgenieResource{Resource: cc.Resource{V: variant{}}}
}

func (r *opsgenieResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &opsgenieResource{}
	_ resource.ResourceWithConfigure   = &opsgenieResource{}
	_ resource.ResourceWithImportState = &opsgenieResource{}
)
