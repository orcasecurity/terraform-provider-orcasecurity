package splunk

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type state struct {
	ID                  types.String `tfsdk:"id"`
	TemplateName        types.String `tfsdk:"template_name"`
	IsEnabled           types.Bool   `tfsdk:"is_enabled"`
	IsDefault           types.Bool   `tfsdk:"is_default"`
	URL                 types.String `tfsdk:"url"`
	Token               types.String `tfsdk:"token"`
	AllowSelfSignedCert types.Bool   `tfsdk:"allow_self_signed_cert"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: types.SetNull(types.StringType)}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_splunk" }
func (variant) UIName() string              { return "Splunk integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage a Splunk HEC integration in Orca. Creates an external service config of `service_name = \"splunk\"`. The HEC token is stored in Orca's secret store and is never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"url": schema.StringAttribute{
			Required:    true,
			Description: "Splunk HEC endpoint URL (for example, `https://prd-p-xxxxx.splunkcloud.com:8088/services/collector/event`).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"token": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Splunk HEC token. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"allow_self_signed_cert": schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether Orca should accept self-signed TLS certificates when calling the Splunk endpoint. Defaults to `false`.",
			Default:     booldefault.StaticBool(false),
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.SplunkExternalServiceConfig {
	return api_client.SplunkExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config: api_client.SplunkConfig{
			URL:                 s.URL.ValueString(),
			Token:               s.Token.ValueString(),
			AllowSelfSignedCert: s.AllowSelfSignedCert.ValueBool(),
		},
	}
}

func apiObj(o *api_client.SplunkExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.URL != "" {
		s.URL = types.StringValue(o.Config.URL)
	}
	s.AllowSelfSignedCert = types.BoolValue(o.Config.AllowSelfSignedCert)
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateSplunkConfig(v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Splunk integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetSplunkConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Splunk integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateSplunkConfig(templateName, v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Splunk integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteSplunkConfig(templateName)
}

type splunkResource struct{ cc.Resource }

func NewSplunkResource() resource.Resource {
	return &splunkResource{Resource: cc.Resource{V: variant{}}}
}

func (r *splunkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &splunkResource{}
	_ resource.ResourceWithConfigure   = &splunkResource{}
	_ resource.ResourceWithImportState = &splunkResource{}
)
