package user

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

type userStateModel struct {
	ID          types.String `tfsdk:"id"`
	Email       types.String `tfsdk:"email"`
	First       types.String `tfsdk:"first"`
	Last        types.String `tfsdk:"last"`
	MFARequired types.Bool   `tfsdk:"mfa_required"`
	MFAEnabled  types.Bool   `tfsdk:"mfa_enabled"`
}

type userDataSource struct {
	apiClient *api_client.APIClient
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

func (ds *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (ds *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "User email address. This is the same email address used for authentication.",
			},
			"first": schema.StringAttribute{
				Computed:    true,
				Description: "First name of the user.",
			},
			"last": schema.StringAttribute{
				Computed:    true,
				Description: "Last name of the user.",
			},
			"mfa_required": schema.BoolAttribute{
				Computed:    true,
				Description: "Is multi-factor authentication required for this user.",
			},
			"mfa_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Is multi-factor authentication enabled for this user.",
			},
		},
	}
}

func (ds *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state userStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	item, err := ds.apiClient.GetUserByEmail(state.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read user", err.Error())
		return
	}

	state.ID = types.StringValue(item.ID)
	state.Email = types.StringValue(item.Email)
	state.First = types.StringValue(item.First)
	state.Last = types.StringValue(item.Last)
	state.MFARequired = types.BoolValue(item.MFARequired)
	state.MFAEnabled = types.BoolValue(item.MFAEnabled)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
