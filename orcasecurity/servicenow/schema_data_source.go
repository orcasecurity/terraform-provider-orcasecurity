package servicenow

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
	_ datasource.DataSource              = &schemaDataSource{}
	_ datasource.DataSourceWithConfigure = &schemaDataSource{}
)

type schemaDataSource struct {
	apiClient *api_client.APIClient
}

type schemaFieldModel struct {
	Element      types.String `tfsdk:"element"`
	Label        types.String `tfsdk:"label"`
	Type         types.String `tfsdk:"type"`
	DefaultValue types.String `tfsdk:"default_value"`
	MaxLength    types.String `tfsdk:"max_length"`
	Choice       types.String `tfsdk:"choice"`
}

type schemaDataSourceModel struct {
	ResourceID types.String       `tfsdk:"resource_id"`
	Type       types.String       `tfsdk:"type"`
	Fields     []schemaFieldModel `tfsdk:"fields"`
	Elements   types.List         `tfsdk:"elements"`
}

func NewServiceNowSchemaDataSource() datasource.DataSource {
	return &schemaDataSource{}
}

func (ds *schemaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_servicenow_schema"
}

func (ds *schemaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *schemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up the ServiceNow field schema for a credentials resource. SIR and ITSM templates share the same `orcasecurity_integration_servicenow_resource` but map to different ServiceNow tables (`sn_si_incident` vs `incident`), so pick the variant with `type`. Use the returned `elements` list to discover which keys are valid in `mapping_json` on the matching template resource. Backs GET /api/resources/{resource_id}/service_now/{type}/schema.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the ServiceNow credentials resource (an `orcasecurity_integration_servicenow_resource` resource ID).",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Which ServiceNow table schema to read: `sir` (Security Incident Response, `sn_si_incident`) or `itsm` (IT Service Management, `incident`).",
				Validators: []validator.String{
					stringvalidator.OneOf("sir", "itsm"),
				},
			},
			"elements": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Flat list of ServiceNow field element names (the keys allowed in `mapping_json`).",
			},
			"fields": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Full schema entries returned by Orca.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"element":       schema.StringAttribute{Computed: true},
						"label":         schema.StringAttribute{Computed: true},
						"type":          schema.StringAttribute{Computed: true},
						"default_value": schema.StringAttribute{Computed: true},
						"max_length":    schema.StringAttribute{Computed: true},
						"choice":        schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (ds *schemaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state schemaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snType := state.Type.ValueString()
	fields, err := ds.apiClient.GetServiceNowSchema(state.ResourceID.ValueString(), snType)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ServiceNow schema",
			fmt.Sprintf("Could not read %s schema for resource %s: %s", snType, state.ResourceID.ValueString(), err.Error()),
		)
		return
	}

	state.Fields = make([]schemaFieldModel, 0, len(fields))
	elementNames := make([]string, 0, len(fields))
	for _, f := range fields {
		state.Fields = append(state.Fields, schemaFieldModel{
			Element:      types.StringValue(f.Element),
			Label:        types.StringValue(f.Label),
			Type:         types.StringValue(f.Type),
			DefaultValue: types.StringValue(f.DefaultValue),
			MaxLength:    types.StringValue(f.MaxLength),
			Choice:       types.StringValue(f.Choice),
		})
		elementNames = append(elementNames, f.Element)
	}

	elements, listDiags := types.ListValueFrom(ctx, types.StringType, elementNames)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Elements = elements

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
