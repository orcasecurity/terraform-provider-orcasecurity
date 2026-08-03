package shift_left_repository

import (
	"maps"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func sharedRepoListAttrs() map[string]dschema.Attribute {
	return map[string]dschema.Attribute{
		"id":                    dschema.StringAttribute{Computed: true, Description: "Orca integrated-repository UUID."},
		"name":                  dschema.StringAttribute{Computed: true},
		"url":                   dschema.StringAttribute{Computed: true},
		"project_id":            dschema.StringAttribute{Computed: true},
		"repository_context_id": dschema.StringAttribute{Computed: true},
		"disabled":              dschema.BoolAttribute{Computed: true},
		"status":                dschema.StringAttribute{Computed: true},
		"integration_status":    dschema.StringAttribute{Computed: true},
	}
}

func sharedRepoListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType, "name": types.StringType, "url": types.StringType,
		"project_id": types.StringType, "repository_context_id": types.StringType,
		"disabled": types.BoolType, "status": types.StringType, "integration_status": types.StringType,
	}
}

func sharedRepoListValues(r api_client.ScmRepository) map[string]attr.Value {
	return map[string]attr.Value{
		"id":                    types.StringValue(r.ID),
		"name":                  types.StringValue(r.RepositoryName),
		"url":                   types.StringValue(r.RepositoryURL),
		"project_id":            tfconv.StringOrNull(r.ProjectID),
		"repository_context_id": tfconv.StringOrNull(r.RepositoryContextID),
		"disabled":              types.BoolValue(r.Disabled),
		"status":                tfconv.StringOrNull(r.Status),
		"integration_status":    tfconv.StringOrNull(r.IntegrationStatus),
	}
}

// mergeMaps combines the shared repo-list maps with per-provider extras; extra wins on conflict.
func mergeMaps[V any](base, extra map[string]V) map[string]V {
	out := make(map[string]V, len(base)+len(extra))
	maps.Copy(out, base)
	maps.Copy(out, extra)
	return out
}
