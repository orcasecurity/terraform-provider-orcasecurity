package webhook

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func CustomHeaderObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"custom": types.StringType,
	}}
}

// CustomHeaderListType is the list-of-objects element type used for the custom_headers map.
func CustomHeaderListType() types.ListType {
	return types.ListType{ElemType: CustomHeaderObjectType()}
}

// CustomHeadersToAPI converts the Terraform Map into the API map shape.
// Returns nil when the map is null/unknown so omitempty drops the field.
func CustomHeadersToAPI(ctx context.Context, headers types.Map) (map[string]api_client.WebhookCustomHeaderValues, diag.Diagnostics) {
	var diags diag.Diagnostics
	if headers.IsNull() || headers.IsUnknown() {
		return nil, diags
	}
	raw := map[string]types.List{}
	diags.Append(headers.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string]api_client.WebhookCustomHeaderValues, len(raw))
	for key, list := range raw {
		if list.IsNull() || list.IsUnknown() {
			continue
		}
		var objs []struct {
			Custom types.String `tfsdk:"custom"`
		}
		diags.Append(list.ElementsAs(ctx, &objs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		values := make(api_client.WebhookCustomHeaderValues, 0, len(objs))
		for _, o := range objs {
			values = append(values, api_client.WebhookCustomHeaderValue{Custom: o.Custom.ValueString()})
		}
		out[key] = values
	}
	return out, diags
}

// CustomHeadersFromAPI converts the API map into a Terraform Map.
// Preserves a null planned value when the API returns no headers.
// A header the API reports as the secret marker is "unchanged": it keeps the list from the
// planned (prior) map, or the marker itself when the prior map has no value for that header.
func CustomHeadersFromAPI(headers map[string]api_client.WebhookCustomHeaderValues, planned types.Map) (types.Map, diag.Diagnostics) {
	listType := CustomHeaderListType()
	if len(headers) == 0 && planned.IsNull() {
		return types.MapNull(listType), nil
	}

	var diags diag.Diagnostics
	elements := make(map[string]attr.Value, len(headers))
	for name, apiValues := range headers {
		if containsSecretMarker(apiValues) {
			if priorList, found := priorHeaderList(planned, name); found {
				elements[name] = priorList
				continue
			}
		}
		list, listDiags := headerValuesToList(apiValues)
		diags.Append(listDiags...)
		elements[name] = list
	}
	if diags.HasError() {
		return types.MapNull(listType), diags
	}
	result, mapDiags := types.MapValue(listType, elements)
	diags.Append(mapDiags...)
	return result, diags
}

func containsSecretMarker(values api_client.WebhookCustomHeaderValues) bool {
	for _, v := range values {
		if v.Custom == api_client.WebhookSecretSetMarker {
			return true
		}
	}
	return false
}

// priorHeaderList returns the list stored in the prior map for a header name.
func priorHeaderList(prior types.Map, name string) (attr.Value, bool) {
	if prior.IsNull() || prior.IsUnknown() {
		return nil, false
	}
	list, found := prior.Elements()[name]
	if !found || list.IsNull() || list.IsUnknown() {
		return nil, false
	}
	return list, true
}

func headerValuesToList(values api_client.WebhookCustomHeaderValues) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := CustomHeaderObjectType()
	objs := make([]attr.Value, 0, len(values))
	for _, v := range values {
		obj, objDiags := types.ObjectValue(objType.AttrTypes, map[string]attr.Value{
			"custom": types.StringValue(v.Custom),
		})
		diags.Append(objDiags...)
		objs = append(objs, obj)
	}
	list, listDiags := types.ListValue(objType, objs)
	diags.Append(listDiags...)
	return list, diags
}

// ExtractTopLevel populates only the cross-cutting identifiers. Used on Create/Update so the
// planned (sensitive) config block survives the Plugin Framework's consistency check.
func ExtractTopLevel(api *api_client.WebhookExternalServiceConfig, _ cc.State, _ *diag.Diagnostics) cc.APIObject {
	return cc.APIObject{
		ID:            api.ID,
		TemplateName:  api.TemplateName,
		IsEnabled:     api.IsEnabled,
		IsDefault:     api.IsDefault,
		BusinessUnits: api.BusinessUnits,
	}
}

// BodyFieldsFromAPI mirrors the API's body_fields list into a Terraform List, preserving the
// planned null-vs-empty shape.
func BodyFieldsFromAPI(ctx context.Context, fields []string, planned types.List) (types.List, diag.Diagnostics) {
	if len(fields) == 0 {
		if planned.IsNull() {
			return types.ListNull(types.StringType), nil
		}
		emptyList, diags := types.ListValue(types.StringType, []attr.Value{})
		return emptyList, diags
	}
	return types.ListValueFrom(ctx, types.StringType, fields)
}
