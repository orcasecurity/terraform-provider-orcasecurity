package akamai

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
	AccessToken  types.String `tfsdk:"access_token"`
	ClientToken  types.String `tfsdk:"client_token"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Host         types.String `tfsdk:"host"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: types.SetNull(types.StringType)}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_akamai" }
func (variant) UIName() string              { return "Akamai integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage an Akamai integration in Orca. Creates an external service config of `service_name = \"akamai\"`. The Akamai credentials (`access_token`, `client_token`, `client_secret`) are stored in Orca's secret store and are never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"access_token": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Akamai EdgeGrid `access_token`.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"client_token": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Akamai EdgeGrid `client_token`.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"client_secret": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Akamai EdgeGrid `client_secret`.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"host": schema.StringAttribute{
			Required:    true,
			Description: "Akamai EdgeGrid host (for example, `akab-xxxxxxxx.luna.akamaiapis.net`).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.AkamaiExternalServiceConfig {
	return api_client.AkamaiExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config: api_client.AkamaiConfig{
			AccessToken:  s.AccessToken.ValueString(),
			ClientToken:  s.ClientToken.ValueString(),
			ClientSecret: s.ClientSecret.ValueString(),
			Host:         s.Host.ValueString(),
		},
	}
}

func apiObj(o *api_client.AkamaiExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.Host != "" {
		s.Host = types.StringValue(o.Config.Host)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateAkamaiConfig(v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Akamai integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetAkamaiConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Akamai integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateAkamaiConfig(templateName, v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Akamai integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteAkamaiConfig(templateName)
}

type akamaiResource struct{ cc.Resource }

func NewAkamaiResource() resource.Resource {
	return &akamaiResource{Resource: cc.Resource{V: variant{}}}
}

func (r *akamaiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &akamaiResource{}
	_ resource.ResourceWithConfigure   = &akamaiResource{}
	_ resource.ResourceWithImportState = &akamaiResource{}
)
