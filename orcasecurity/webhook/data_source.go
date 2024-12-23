package webhook

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &dataSource{}
	_ datasource.DataSourceWithConfigure = &dataSource{}
)

type stateModel struct {
	ID types.String `tfsdk:"id"`

	Config       *config      `tfsdk:"config"`
	CreatedAt    types.String `tfsdk:"created_at"`
	IsEnabled    types.Bool   `tfsdk:"is_enabled"`
	TemplateName types.String `tfsdk:"name"`
}

type config struct {
	CustomHeaders types.Map    `tfsdk:"custom_headers"`
	Type          types.String `tfsdk:"type"`
	WebhookUrl    types.String `tfsdk:"webhook_url"`
}

type dataSource struct {
	apiClient *api_client.APIClient
}

func NewWebhookDataSource() datasource.DataSource {
	return &dataSource{}
}

func (ds *dataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *dataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (ds *dataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source allows you to pull data about a given webhook based on its name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-friendly name that you've selected for the Webhook.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the webhook was created (in Unix timestamp-like format).",
			},
			"is_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the webhook is enabled or not.",
			},
			"config": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"custom_headers": schema.MapAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "They key-value pairs that are the HTTP headers sent by Orca to the webhook URL.",
					},
					"type": schema.StringAttribute{
						Computed: true,
					},
					"webhook_url": schema.StringAttribute{
						Computed:    true,
						Description: "The URL of the webhook.",
					},
				},
				Computed: true,
			},
		},
	}
}

func (ds *dataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state stateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	//item, err := ds.apiClient.GetWebhookByName(req.Config.Raw.String())
	item, err := ds.apiClient.GetWebhookByName(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Webhook", err.Error())
		return
	}

	// Convert the map[string]string to types.Map
	customHeadersMap, diags := types.MapValueFrom(
		ctx,
		types.StringType, // Changed: now expecting map[string]string
		item.Config.CustomHeaders,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(item.ID)
	state.TemplateName = types.StringValue(item.TemplateName)
	state.IsEnabled = types.BoolValue(item.IsEnabled)
	state.Config = &config{
		Type:          types.StringValue(item.Config.Type),
		WebhookUrl:    types.StringValue(item.Config.WebhookUrl),
		CustomHeaders: customHeadersMap,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
