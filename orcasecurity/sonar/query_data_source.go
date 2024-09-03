package sonar

import (
	"context"
	"encoding/json"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &sonarQueryDataSource{}
	_ datasource.DataSourceWithConfigure = &sonarQueryDataSource{}
)

type sonarQueryDataSource struct {
	apiClient *api_client.APIClient
}

type sonarQueryRowStateModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
	Data types.String `tfsdk:"data"`
}

type sonarQueryFilterModel struct {
	Query        types.String `tfsdk:"query"`
	Results      types.List   `tfsdk:"results"`
	Limit        types.Int64  `tfsdk:"limit"`
	StartAtIndex types.Int64  `tfsdk:"start_at_index"`
}

func NewSonarQueryDataSource() datasource.DataSource {
	return &sonarQueryDataSource{}
}

func (ds *sonarQueryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *sonarQueryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_discovery_results"
}

func (ds *sonarQueryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Execute a query and get discovery results.",
		Attributes: map[string]schema.Attribute{
			"limit": schema.Int64Attribute{
				Required:    true,
				Description: "Result set limit",
			},
			"start_at_index": schema.Int64Attribute{
				Required:    true,
				Description: "Index to start results at",
			},
			"query": schema.StringAttribute{
				Description: "The query.",
				Required:    true,
			},
			"results": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required: true,
						},
						"name": schema.StringAttribute{
							Required: true,
						},
						"type": schema.StringAttribute{
							Required: true,
						},
						"data": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
		},
	}
}

func (ds *sonarQueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	//state is of type SonarQueryFilterModel w/ Query, Results, Limit, StartAtIndex as fields
	var state sonarQueryFilterModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	//queryString is stored in state
	queryString := state.Query.ValueString()
	query := make(map[string]interface{})
	err := json.Unmarshal([]byte(queryString), &query)
	if err != nil {
		resp.Diagnostics.AddError("Query decode error", err.Error())
		return
	}

	limit := state.Limit.ValueInt64()
	startIndex := state.StartAtIndex.ValueInt64()
	items, err := ds.apiClient.ExecuteSonarQuery(query, limit, startIndex)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read query results.", err.Error())
		return
	}

	var stateRows []sonarQueryRowStateModel
	for _, row := range items {
		dataAsString, err := json.Marshal(row)
		if err != nil {
			resp.Diagnostics.AddError("Response encode error", err.Error())
			return
		}
		stateRows = append(stateRows, sonarQueryRowStateModel{
			ID:   types.StringValue(row.ID),
			Name: types.StringValue(row.Name),
			Type: types.StringValue(row.Type),
			Data: types.StringValue(string(dataAsString)),
		})

	}

	resultObjAttrs := map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"type": types.StringType,
		"data": types.StringType,
	}
	resultType := types.ObjectType{}.WithAttributeTypes(resultObjAttrs)
	results, diags := types.ListValueFrom(ctx, resultType, stateRows)
	if diags != nil {
		resp.Diagnostics.Append(diags...)
		return
	}
	state.Results = results

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
