package azure_sentinel

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
	LogType       types.String `tfsdk:"log_type"`
	PrimaryKey    types.String `tfsdk:"primary_key"`
	WorkspaceID   types.String `tfsdk:"workspace_id"`
}

func (s *state) GetCommon() *cc.Common {
	return &cc.Common{ID: s.ID, TemplateName: s.TemplateName, IsEnabled: s.IsEnabled, IsDefault: s.IsDefault, BusinessUnits: s.BusinessUnits}
}
func (s *state) SetCommon(c cc.Common) {
	s.ID, s.TemplateName, s.IsEnabled, s.IsDefault, s.BusinessUnits = c.ID, c.TemplateName, c.IsEnabled, c.IsDefault, c.BusinessUnits
}

type variant struct{}

func (variant) TypeNameSuffix() string      { return "_integration_azure_sentinel" }
func (variant) UIName() string              { return "Azure Sentinel integration" }
func (variant) SupportsBusinessUnits() bool { return true }
func (variant) Description() string {
	return "Manage an Azure Sentinel integration in Orca. Creates an external service config of `service_name = \"azure_sentinel\"`. The Log Analytics workspace primary key is stored in Orca's secret store and is never returned by the API."
}
func (variant) VariantAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"log_type": schema.StringAttribute{
			Required:    true,
			Description: "Custom log type name used in the Log Analytics workspace (for example, `OrcaAlerts`).",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"workspace_id": schema.StringAttribute{
			Required:    true,
			Description: "Azure Log Analytics workspace ID that backs the Sentinel instance.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"primary_key": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Azure Log Analytics workspace primary key. Stored in Orca's secret store; never returned by the API.",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
	}
}
func (variant) NewState() cc.State { return &state{} }

func (variant) buildPayload(ctx context.Context, s *state, diags *diag.Diagnostics) api_client.AzureSentinelExternalServiceConfig {
	return api_client.AzureSentinelExternalServiceConfig{
		TemplateName:  s.TemplateName.ValueString(),
		IsEnabled:     s.IsEnabled.ValueBool(),
		IsDefault:     s.IsDefault.ValueBool(),
		BusinessUnits: common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags),
		Config: api_client.AzureSentinelConfig{
			LogType:     s.LogType.ValueString(),
			PrimaryKey:  s.PrimaryKey.ValueString(),
			WorkspaceID: s.WorkspaceID.ValueString(),
		},
	}
}

func apiObj(o *api_client.AzureSentinelExternalServiceConfig, s *state) cc.APIObject {
	if o.Config.LogType != "" {
		s.LogType = types.StringValue(o.Config.LogType)
	}
	if o.Config.WorkspaceID != "" {
		s.WorkspaceID = types.StringValue(o.Config.WorkspaceID)
	}
	return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault, BusinessUnits: o.BusinessUnits}
}

func (v variant) Create(c *api_client.APIClient, ctx context.Context, plan cc.State, diags *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	created, err := c.CreateAzureSentinelConfig(v.buildPayload(ctx, s, diags))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error creating Azure Sentinel integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(created, s), nil
}

func (v variant) Read(c *api_client.APIClient, _ context.Context, st cc.State, _ *diag.Diagnostics) (cc.APIObject, bool, diag.Diagnostics) {
	s := st.(*state)
	current, err := c.GetAzureSentinelConfig(s.TemplateName.ValueString())
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error reading Azure Sentinel integration", err.Error())
		return cc.APIObject{}, false, d
	}
	if current == nil {
		return cc.APIObject{}, false, nil
	}
	return apiObj(current, s), true, nil
}

func (v variant) Update(c *api_client.APIClient, ctx context.Context, plan cc.State, templateName string, diags *diag.Diagnostics) (cc.APIObject, diag.Diagnostics) {
	s := plan.(*state)
	updated, err := c.UpdateAzureSentinelConfig(templateName, v.buildPayload(ctx, s, diags))
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error updating Azure Sentinel integration", err.Error())
		return cc.APIObject{}, d
	}
	return apiObj(updated, s), nil
}

func (v variant) Delete(c *api_client.APIClient, templateName string) error {
	return c.DeleteAzureSentinelConfig(templateName)
}

type azureSentinelResource struct{ cc.Resource }

func NewAzureSentinelResource() resource.Resource {
	return &azureSentinelResource{Resource: cc.Resource{V: variant{}}}
}

func (r *azureSentinelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = cc.Schema(r.V)
}

var (
	_ resource.Resource                = &azureSentinelResource{}
	_ resource.ResourceWithConfigure   = &azureSentinelResource{}
	_ resource.ResourceWithImportState = &azureSentinelResource{}
)
