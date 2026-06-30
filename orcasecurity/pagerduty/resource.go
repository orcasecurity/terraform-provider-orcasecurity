package pagerduty

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	ID             types.String `tfsdk:"id"`
	TemplateName   types.String `tfsdk:"template_name"`
	IsEnabled      types.Bool   `tfsdk:"is_enabled"`
	IsDefault      types.Bool   `tfsdk:"is_default"`
	IntegrationKey types.String `tfsdk:"integration_key"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{
		ID:            s.ID,
		TemplateName:  s.TemplateName,
		IsEnabled:     s.IsEnabled,
		IsDefault:     s.IsDefault,
		BusinessUnits: types.SetNull(types.StringType),
	}
}

func (s *state) SetCommon(c cc.Common) {
	s.ID = c.ID
	s.TemplateName = c.TemplateName
	s.IsEnabled = c.IsEnabled
	s.IsDefault = c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_pagerduty" }
func (variant) UIName() string              { return "PagerDuty integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage a PagerDuty integration in Orca. Creates an external service config of `service_name = \"pagerduty\"` and stores the integration key in Orca's secret store."
}

func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"integration_key": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "PagerDuty Events API V2 integration key. Stored in Orca's secret store; never returned by the API.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
	}
}

func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.PagerDutyExternalServiceConfig {
	return api_client.PagerDutyExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config: api_client.PagerDutyConfig{
			IntegrationKey: s.IntegrationKey.ValueString(),
		},
	}
}

func apiObj(o *api_client.PagerDutyExternalServiceConfig) cc.APIObject {
	return cc.APIObject{
		ID:           o.ID,
		TemplateName: o.TemplateName,
		IsEnabled:    o.IsEnabled,
		IsDefault:    o.IsDefault,
	}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	created, err := c.CreatePagerDutyConfig(v.buildPayload(plan.(*state)))
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error creating PagerDuty integration", err.Error())
		return cc.APIObject{}, diags
	}
	return apiObj(created), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, state cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	current, err := c.GetPagerDutyConfig(state.GetCommon().TemplateName.ValueString())
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error reading PagerDuty integration", err.Error())
		return cc.APIObject{}, false, diags
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	updated, err := c.UpdatePagerDutyConfig(templateName, v.buildPayload(plan.(*state)))
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error updating PagerDuty integration", err.Error())
		return cc.APIObject{}, diags
	}
	return apiObj(updated), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeletePagerDutyConfig(templateName)
}

type pagerDutyResource struct {
	cc.Resource
}

func NewPagerDutyResource() resource.Resource {
	return &pagerDutyResource{Resource: cc.Resource{V: variant{}}}
}

func (r *pagerDutyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &pagerDutyResource{}
	_ resource.ResourceWithConfigure   = &pagerDutyResource{}
	_ resource.ResourceWithImportState = &pagerDutyResource{}
)
