package automation_v2_priorities

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &automationPrioritiesDataSource{}
	_ datasource.DataSourceWithConfigure = &automationPrioritiesDataSource{}
)

type automationPrioritiesDataSource struct {
	apiClient *api_client.APIClient
}

type automationPriorityEntryModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Priority types.Int64  `tfsdk:"priority"`
	Status   types.String `tfsdk:"status"`
}

type automationPrioritiesModel struct {
	Automations []automationPriorityEntryModel `tfsdk:"automations"`
}

func NewAutomationPrioritiesDataSource() datasource.DataSource {
	return &automationPrioritiesDataSource{}
}

func (ds *automationPrioritiesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *automationPrioritiesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automation_v2_priorities"
}

func (ds *automationPrioritiesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the automations visible to the API token (the list is business-unit/RBAC scoped) " +
			"in evaluation order (priority ascending), including automations not managed by Terraform.",
		Attributes: map[string]schema.Attribute{
			"automations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Automations in evaluation order. Priorities may contain duplicates or gaps from legacy data.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Automation ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Automation name.",
						},
						"priority": schema.Int64Attribute{
							Computed:    true,
							Description: "Evaluation-order priority (1 = evaluated first).",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "Automation status.",
						},
					},
				},
			},
		},
	}
}

// fetchAutomations retrieves all automations in server evaluation order and
// maps them to the data source's entry model.
func (ds *automationPrioritiesDataSource) fetchAutomations() ([]automationPriorityEntryModel, error) {
	instances, err := ds.apiClient.ListAutomationsV2()
	if err != nil {
		return nil, err
	}
	entries := make([]automationPriorityEntryModel, 0, len(instances))
	for _, instance := range instances {
		entries = append(entries, automationPriorityEntryModel{
			ID:       types.StringValue(instance.ID),
			Name:     types.StringValue(instance.Name),
			Priority: types.Int64PointerValue(instance.Priority),
			Status:   types.StringValue(instance.Status),
		})
	}
	return entries, nil
}

func (ds *automationPrioritiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	entries, err := ds.fetchAutomations()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading automations",
			"Could not list automations: "+err.Error(),
		)
		return
	}
	state := automationPrioritiesModel{Automations: entries}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
