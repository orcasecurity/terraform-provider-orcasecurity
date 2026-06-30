package snyk

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

var snykRegions = []string{"US", "EU", "AU", "US2"}

type state struct {
	ID           types.String `tfsdk:"id"`
	TemplateName types.String `tfsdk:"template_name"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	IsDefault    types.Bool   `tfsdk:"is_default"`
	APIToken     types.String `tfsdk:"api_token"`
	Region       types.String `tfsdk:"region"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: types.SetNull(types.StringType)}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_snyk" }
func (variant) UIName() string              { return "Snyk integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage a Snyk integration in Orca. Creates an external service config of `service_name = \"snyk\"`. The Snyk API token is stored in Orca's secret store and is never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_token": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Snyk service account API token. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"region": schema.StringAttribute{
			Required: true,
			Description: "Snyk tenant region. The Orca UI exposes four choices — pass the matching API code here:\n" +
				"  - `US`  — United States (app.snyk.io)\n" +
				"  - `US2` — United States 2 (app.us.snyk.io)\n" +
				"  - `EU`  — European Union (app.eu.snyk.io)\n" +
				"  - `AU`  — Australia (app.au.snyk.io)",
			Validators: []validator.String{stringvalidator.OneOf(snykRegions...)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.SnykExternalServiceConfig {
	return api_client.SnykExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config:       api_client.SnykConfig{APIToken: s.APIToken.ValueString(), Region: s.Region.ValueString()},
	}
}

func apiObj(o *api_client.SnykExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.Region != "" {
		s.Region = types.StringValue(o.Config.Region)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateSnykConfig(v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Snyk integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetSnykConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Snyk integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateSnykConfig(templateName, v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Snyk integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteSnykConfig(templateName)
}

type snykResource struct{ cc.Resource }

func NewSnykResource() resource.Resource {
	return &snykResource{Resource: cc.Resource{V: variant{}}}
}

func (r *snykResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &snykResource{}
	_ resource.ResourceWithConfigure   = &snykResource{}
	_ resource.ResourceWithImportState = &snykResource{}
)
