package business_unit

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &businessUnitDataSource{}
	_ datasource.DataSourceWithConfigure = &businessUnitDataSource{}
)

type businessUnitStateModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Filter          types.Object `tfsdk:"filter"`
	ShiftLeftFilter types.Object `tfsdk:"shift_left_filter"`
}

type businessUnitDataSource struct {
	apiClient *api_client.APIClient
}

func NewBusinessUnitDataSource() datasource.DataSource {
	return &businessUnitDataSource{}
}

func (ds *businessUnitDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *businessUnitDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_business_unit"
}

func (ds *businessUnitDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch business unit by ID or name. Exactly one of 'id' or 'name' must be specified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Business Unit ID. Cannot be used with 'name'.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Business Unit name. Cannot be used with 'id'.",
			},
			"filter": schema.ObjectAttribute{
				Computed: true,
				AttributeTypes: map[string]attr.Type{
					"cloud_providers": types.SetType{ElemType: types.StringType},
					"custom_tags":     types.SetType{ElemType: types.StringType},
					"cloud_tags":      types.SetType{ElemType: types.StringType},
					"account_tags":    types.SetType{ElemType: types.StringType},
					"cloud_accounts":  types.SetType{ElemType: types.StringType},
				},
				Description: "Filter configuration for the business unit.",
			},
			"shift_left_filter": schema.ObjectAttribute{
				Computed: true,
				AttributeTypes: map[string]attr.Type{
					"shift_left_projects": types.SetType{ElemType: types.StringType},
				},
				Description: "Shift left filter configuration for the business unit.",
			},
		},
	}
}

func (ds *businessUnitDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state businessUnitStateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one of id or name is provided
	hasID := !state.ID.IsNull() && !state.ID.IsUnknown()
	hasName := !state.Name.IsNull() && !state.Name.IsUnknown()

	if hasID && hasName {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Conflicting configuration",
			"Cannot specify both 'id' and 'name'. Please specify exactly one.",
		)
		return
	}

	if !hasID && !hasName {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Missing configuration",
			"Must specify either 'id' or 'name'.",
		)
		return
	}

	var item *api_client.BusinessUnit
	var err error

	if hasID {
		item, err = ds.apiClient.GetBusinessUnit(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read business unit", err.Error())
			return
		}
		if item == nil {
			resp.Diagnostics.AddError("Business unit not found", "Business unit with ID '"+state.ID.ValueString()+"' does not exist")
			return
		}
		// Set the name from the API response
		state.Name = types.StringValue(item.Name)
	} else {
		item, err = ds.apiClient.GetBusinessUnitByName(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read business unit", err.Error())
			return
		}
		// Set the ID from the API response
		state.ID = types.StringValue(item.ID)
	}

	// Define the attribute types for filter object
	filterAttributeTypes := map[string]attr.Type{
		"cloud_providers": types.SetType{ElemType: types.StringType},
		"custom_tags":     types.SetType{ElemType: types.StringType},
		"cloud_tags":      types.SetType{ElemType: types.StringType},
		"account_tags":    types.SetType{ElemType: types.StringType},
		"cloud_accounts":  types.SetType{ElemType: types.StringType},
	}

	// Convert filter to object
	filterAttrs := make(map[string]attr.Value)
	if item.Filter != nil {
		cloudProviders, diags := types.SetValueFrom(ctx, types.StringType, item.Filter.CloudProviders)
		resp.Diagnostics.Append(diags...)
		filterAttrs["cloud_providers"] = cloudProviders

		customTags, diags := types.SetValueFrom(ctx, types.StringType, item.Filter.CustomTags)
		resp.Diagnostics.Append(diags...)
		filterAttrs["custom_tags"] = customTags

		cloudTags, diags := types.SetValueFrom(ctx, types.StringType, item.Filter.CloudTags)
		resp.Diagnostics.Append(diags...)
		filterAttrs["cloud_tags"] = cloudTags

		accountTags, diags := types.SetValueFrom(ctx, types.StringType, item.Filter.AccountTags)
		resp.Diagnostics.Append(diags...)
		filterAttrs["account_tags"] = accountTags

		cloudAccounts, diags := types.SetValueFrom(ctx, types.StringType, item.Filter.CloudAccounts)
		resp.Diagnostics.Append(diags...)
		filterAttrs["cloud_accounts"] = cloudAccounts
	} else {
		// Create empty sets for null filter
		emptySet, diags := types.SetValueFrom(ctx, types.StringType, []string{})
		resp.Diagnostics.Append(diags...)
		filterAttrs["cloud_providers"] = emptySet
		filterAttrs["custom_tags"] = emptySet
		filterAttrs["cloud_tags"] = emptySet
		filterAttrs["account_tags"] = emptySet
		filterAttrs["cloud_accounts"] = emptySet
	}

	// Use types.ObjectValue instead of types.ObjectValueFrom
	filterObject, diags := types.ObjectValue(filterAttributeTypes, filterAttrs)
	resp.Diagnostics.Append(diags...)
	state.Filter = filterObject

	// Define the attribute types for shift left filter object
	shiftLeftFilterAttributeTypes := map[string]attr.Type{
		"shift_left_projects": types.SetType{ElemType: types.StringType},
	}

	// Convert shift left filter to object
	shiftLeftFilterAttrs := make(map[string]attr.Value)
	if item.ShiftLeftFilter != nil {
		shiftLeftProjects, diags := types.SetValueFrom(ctx, types.StringType, item.ShiftLeftFilter.ShiftLeftProjects)
		resp.Diagnostics.Append(diags...)
		shiftLeftFilterAttrs["shift_left_projects"] = shiftLeftProjects
	} else {
		// Create empty set for null shift left filter
		emptySet, diags := types.SetValueFrom(ctx, types.StringType, []string{})
		resp.Diagnostics.Append(diags...)
		shiftLeftFilterAttrs["shift_left_projects"] = emptySet
	}

	// Use types.ObjectValue instead of types.ObjectValueFrom
	shiftLeftFilterObject, diags := types.ObjectValue(shiftLeftFilterAttributeTypes, shiftLeftFilterAttrs)
	resp.Diagnostics.Append(diags...)
	state.ShiftLeftFilter = shiftLeftFilterObject

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
