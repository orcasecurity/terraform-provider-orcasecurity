package zscaler

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
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	VanityDomain types.String `tfsdk:"vanity_domain"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: types.SetNull(types.StringType)}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_zscaler_zpa" }
func (variant) UIName() string              { return "Zscaler ZPA integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage a Zscaler ZPA integration in Orca. Creates an external service config of `service_name = \"zscaler\"`. The OAuth `client_id` and `client_secret` are stored in Orca's secret store and are never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"vanity_domain": schema.StringAttribute{
			Required:    true,
			Description: "Zscaler ZPA vanity domain (the customer-specific tenant identifier used in the OAuth URL).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"client_id": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Zscaler ZPA OAuth `client_id`. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"client_secret": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Zscaler ZPA OAuth `client_secret`. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.ZscalerExternalServiceConfig {
	return api_client.ZscalerExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config: api_client.ZscalerConfig{
			VanityDomain: s.VanityDomain.ValueString(),
			ClientID:     s.ClientID.ValueString(),
			ClientSecret: s.ClientSecret.ValueString(),
		},
	}
}

func apiObj(o *api_client.ZscalerExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.VanityDomain != "" {
		s.VanityDomain = types.StringValue(o.Config.VanityDomain)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateZscalerConfig(v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Zscaler ZPA integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetZscalerConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Zscaler ZPA integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateZscalerConfig(templateName, v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Zscaler ZPA integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteZscalerConfig(templateName)
}

type zscalerResource struct{ cc.Resource }

func NewZscalerResource() resource.Resource {
	return &zscalerResource{Resource: cc.Resource{V: variant{}}}
}

func (r *zscalerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &zscalerResource{}
	_ resource.ResourceWithConfigure   = &zscalerResource{}
	_ resource.ResourceWithImportState = &zscalerResource{}
)
