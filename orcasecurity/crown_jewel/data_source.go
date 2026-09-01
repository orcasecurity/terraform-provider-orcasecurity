package crown_jewel

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &crownJewelDataSource{}
	_ datasource.DataSourceWithConfigure = &crownJewelDataSource{}
)

type crownJewelDataSource struct {
	apiClient *api_client.APIClient
}

func NewCrownJewelDataSource() datasource.DataSource {
	return &crownJewelDataSource{}
}

func (d *crownJewelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_crown_jewel"
}

func (d *crownJewelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (d *crownJewelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a user-defined (user-marked) crown jewel by `group_unique_id`. " +
			"Useful to read the current Reason before import, or to confirm an asset is marked. " +
			"Returns an error when the asset is not user-marked.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as `group_unique_id`.",
			},
			"group_unique_id": schema.StringAttribute{
				Required:    true,
				Description: "Inventory group unique id of the asset to look up.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Computed: true,
				Description: "Reason on the user-marked crown jewel — the same field as **Reason** in the Orca UI " +
					"(\"Mark as Crown Jewel\").",
			},
		},
	}
}

func (d *crownJewelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config stateModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	gid := config.GroupUniqueID.ValueString()
	instance, err := d.apiClient.GetCrownJewel(gid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading crown jewel",
			fmt.Sprintf("Could not read crown jewel %s: %s", gid, err.Error()),
		)
		return
	}
	if instance == nil {
		resp.Diagnostics.AddError(
			"Crown jewel not found",
			fmt.Sprintf("No user-marked crown jewel exists for group_unique_id %q.", gid),
		)
		return
	}

	state := stateModel{
		ID:            types.StringValue(instance.GroupUniqueID),
		GroupUniqueID: types.StringValue(instance.GroupUniqueID),
		Description:   types.StringValue(instance.Description),
	}
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
