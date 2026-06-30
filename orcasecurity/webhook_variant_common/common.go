// Package webhook_variant_common builds the Spec[WebhookExternalServiceConfig] that Coralogix,
// Torq, and future webhook variants share. Orca stores them under service_name="webhook" with
// config.type pinned to the variant name; the per-variant resource file just calls NewResource
// with its type-name suffix, description, and api_client method refs. The CRUD loop lives in
// config_integration_common.Spec[P] alongside every other config integration.
package webhook_variant_common

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	common "terraform-provider-orcasecurity/orcasecurity/integrations_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// state is the shared TF model used by every webhook variant. CommonFieldsWithBU brings in
// the id / template_name / is_enabled / is_default / business_units quartet so we don't
// declare them per variant.
type state struct {
	cc.CommonFieldsWithBU
	WebhookURL    types.String `tfsdk:"webhook_url"`
	APIKey        types.String `tfsdk:"api_key"`
	BodyFields    types.List   `tfsdk:"body_fields"`
	CustomHeaders types.Map    `tfsdk:"custom_headers"`
}

// Options is what each per-variant package supplies — the only delta between Coralogix and
// Torq (other than naming and prose).
type Options struct {
	TypeNameSuffix    string
	UIName            string
	Description       string
	URLDescription    string
	APIKeyDescription string
	Create            func(*api_client.APIClient, api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error)
	Get               func(*api_client.APIClient, string) (*api_client.WebhookExternalServiceConfig, error)
	Update            func(*api_client.APIClient, string, api_client.WebhookExternalServiceConfig) (*api_client.WebhookExternalServiceConfig, error)
	Delete            func(*api_client.APIClient, string) error
}

func CustomHeaderObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"custom": types.StringType,
	}}
}

func customHeaderListType() types.ListType {
	return types.ListType{ElemType: CustomHeaderObjectType()}
}

func variantAttributes(opts Options) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"webhook_url": schema.StringAttribute{
			Required:    true,
			Description: opts.URLDescription,
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"api_key": schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: opts.APIKeyDescription,
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"body_fields": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Optional list of Orca alert fields to include in the request body.",
		},
		"custom_headers": schema.MapAttribute{
			Optional:    true,
			ElementType: customHeaderListType(),
			Description: "Optional custom HTTP headers, keyed by header name. Each value is a list of `{ custom = \"<value>\" }` objects so a single header can carry multiple values.",
		},
	}
}

// CustomHeadersToAPI is exported so unit tests can exercise the Terraform Map → API map
// conversion. Returns nil when the map is null/unknown so omitempty drops the field.
func CustomHeadersToAPI(ctx context.Context, headers types.Map) (map[string][]api_client.WebhookCustomHeaderValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	if headers.IsNull() || headers.IsUnknown() {
		return nil, diags
	}
	raw := map[string]types.List{}
	diags.Append(headers.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}

	out := make(map[string][]api_client.WebhookCustomHeaderValue, len(raw))
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
		values := make([]api_client.WebhookCustomHeaderValue, 0, len(objs))
		for _, o := range objs {
			values = append(values, api_client.WebhookCustomHeaderValue{Custom: o.Custom.ValueString()})
		}
		out[key] = values
	}
	return out, diags
}

// CustomHeadersFromAPI is exported so unit tests can exercise the API map → Terraform Map
// conversion. Preserves a null planned value when the API returns no headers.
func CustomHeadersFromAPI(headers map[string][]api_client.WebhookCustomHeaderValue, planned types.Map) (types.Map, diag.Diagnostics) {
	listType := customHeaderListType()
	if len(headers) == 0 && planned.IsNull() {
		return types.MapNull(listType), nil
	}

	objType := CustomHeaderObjectType()
	elements := make(map[string]attr.Value, len(headers))
	var diags diag.Diagnostics
	for key, values := range headers {
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
		elements[key] = list
	}
	if diags.HasError() {
		return types.MapNull(listType), diags
	}
	result, mapDiags := types.MapValue(listType, elements)
	diags.Append(mapDiags...)
	return result, diags
}

// extractTopLevel populates only the cross-variant identifiers. Used on Create/Update so the
// planned (sensitive) config block survives the Plugin Framework's consistency check.
func extractTopLevel(api *api_client.WebhookExternalServiceConfig, _ cc.State) cc.APIObject {
	return cc.APIObject{
		ID:            api.ID,
		TemplateName:  api.TemplateName,
		IsEnabled:     api.IsEnabled,
		IsDefault:     api.IsDefault,
		BusinessUnits: api.BusinessUnits,
	}
}

// extractFull refreshes the whole config block from the API response. Used on Read so the
// next plan can detect drift. api_key is intentionally not overwritten with API echoes — the
// API may strip or re-encode the value, and the user-supplied secret already lives in state.
func extractFull(api *api_client.WebhookExternalServiceConfig, st cc.State) cc.APIObject {
	s := st.(*state)
	s.WebhookURL = types.StringValue(api.Config.WebhookURL)
	if s.APIKey.IsUnknown() {
		if api.Config.APIKey != "" {
			s.APIKey = types.StringValue(api.Config.APIKey)
		} else {
			s.APIKey = types.StringNull()
		}
	}
	bodyFields, _ := bodyFieldsFromAPI(context.Background(), api.Config.BodyFields, s.BodyFields)
	s.BodyFields = bodyFields
	headers, _ := CustomHeadersFromAPI(api.Config.CustomHeaders, s.CustomHeaders)
	s.CustomHeaders = headers
	return extractTopLevel(api, st)
}

func bodyFieldsFromAPI(ctx context.Context, fields []string, planned types.List) (types.List, diag.Diagnostics) {
	if len(fields) == 0 {
		if planned.IsNull() {
			return types.ListNull(types.StringType), nil
		}
		emptyList, diags := types.ListValue(types.StringType, []attr.Value{})
		return emptyList, diags
	}
	return types.ListValueFrom(ctx, types.StringType, fields)
}

// NewResource returns the resource.Resource for a webhook variant. The Spec[P] CRUD loop is
// shared with every other config integration; the only webhook-specific piece is the
// Read-vs-Create/Update refresh split, expressed via ExtractOnRead.
//
// Why the split: the Plugin Framework complains "inconsistent sensitive attribute" whenever
// a parent block containing a Sensitive attribute (api_key) shows a diff after apply.
// On Create/Update we leave the planned config exactly as the user submitted it; on Read
// we refresh the whole config block from the API so subsequent `terraform plan` calls can
// detect drift.
func NewResource(opts Options) resource.Resource {
	return cc.New(cc.Spec[api_client.WebhookExternalServiceConfig]{
		TypeNameSuffix:        opts.TypeNameSuffix,
		UIName:                opts.UIName,
		Description:           opts.Description,
		SupportsBusinessUnits: true,
		VariantAttributes:     variantAttributes(opts),
		NewState:              func() cc.State { return &state{} },
		BuildPayload: func(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.WebhookExternalServiceConfig {
			s := st.(*state)
			payload := api_client.WebhookExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config: api_client.WebhookResourceConfig{
					WebhookURL: s.WebhookURL.ValueString(),
					APIKey:     s.APIKey.ValueString(),
				},
			}
			if !s.BodyFields.IsNull() && !s.BodyFields.IsUnknown() {
				var fields []string
				diags.Append(s.BodyFields.ElementsAs(ctx, &fields, false)...)
				payload.Config.BodyFields = fields
			}
			headers, headerDiags := CustomHeadersToAPI(ctx, s.CustomHeaders)
			diags.Append(headerDiags...)
			payload.Config.CustomHeaders = headers
			payload.BusinessUnits = common.BusinessUnitsToAPI(ctx, s.BusinessUnits, diags)
			return payload
		},
		// Extract: Create/Update path. Touches only top-level fields so the planned (sensitive)
		// config block survives the post-apply consistency check.
		Extract:       extractTopLevel,
		// ExtractOnRead: Read path. Refreshes the whole resource so drift in any non-secret
		// config field is detected on the next plan. api_key is intentionally NOT overwritten —
		// the API may strip or re-encode the value and we already keep the user-supplied secret
		// in state from Create/Update.
		ExtractOnRead: extractFull,
		Create: opts.Create,
		Get:    opts.Get,
		Update: opts.Update,
		Delete: opts.Delete,
	})
}
