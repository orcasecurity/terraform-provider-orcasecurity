package terraform_cloud

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
	APIToken     types.String `tfsdk:"api_token"`
	APIURL       types.String `tfsdk:"api_url"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: types.SetNull(types.StringType)}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_terraform_cloud" }
func (variant) UIName() string              { return "Terraform Cloud integration" }
func (variant) SupportsBusinessUnits() bool { return false }
func (variant) Description() string {
	return "Manage a Terraform Cloud (HCP Terraform) integration in Orca. Creates an external service config of `service_name = \"terraform_cloud\"`. The API token is stored in Orca's secret store and is never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_token": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Terraform Cloud service-account API token. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"api_url": schema.StringAttribute{
			Required:    true,
			Description: "Terraform Cloud API URL. Use `https://app.terraform.io` for HCP Terraform or your Terraform Enterprise hostname.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(s *state) api_client.TerraformCloudExternalServiceConfig {
	return api_client.TerraformCloudExternalServiceConfig{
		TemplateName: s.TemplateName.ValueString(),
		IsEnabled:    s.IsEnabled.ValueBool(),
		IsDefault:    s.IsDefault.ValueBool(),
		Config:       api_client.TerraformCloudConfig{APIToken: s.APIToken.ValueString(), APIURL: s.APIURL.ValueString()},
	}
}

func apiObj(o *api_client.TerraformCloudExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.APIURL != "" {
		s.APIURL = types.StringValue(o.Config.APIURL)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
}

func (v variant) Create(c *api_client.APIClient, _ context.Context, plan cc.State, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateTerraformCloudConfig(v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Terraform Cloud integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetTerraformCloudConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Terraform Cloud integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, _ context.Context, plan cc.State, templateName string, _ *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateTerraformCloudConfig(templateName, v.buildPayload(s))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Terraform Cloud integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteTerraformCloudConfig(templateName)
}

type tfcResource struct{ cc.Resource }

func NewTerraformCloudResource() resource.Resource {
	return &tfcResource{Resource: cc.Resource{V: variant{}}}
}

func (r *tfcResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &tfcResource{}
	_ resource.ResourceWithConfigure   = &tfcResource{}
	_ resource.ResourceWithImportState = &tfcResource{}
)
