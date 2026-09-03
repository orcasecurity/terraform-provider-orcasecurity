package shift_left_policy

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateConfig_ScmPostureRejectsProjectsIds(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("scm_posture"),
		Name:                     types.StringValue("scm"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("proj-1"),
		}),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for scm_posture + projects_ids")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "projects_ids") || !strings.Contains(detail, "scm_posture") {
		t.Fatalf("unexpected detail: %s", detail)
	}
}

func TestValidateConfig_ScmPostureAllowsOmittedProjectsIds(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("scm_posture"),
		Name:                     types.StringValue("scm"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ProjectsIds:              types.ListNull(types.StringType),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error when projects_ids omitted: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_LicensesAllowsProjectsIds(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("licenses"),
		Name:                     types.StringValue("lic"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("proj-1"),
		}),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error for licenses + projects_ids: %v", resp.Diagnostics)
	}
}

func validateConfig(t *testing.T, model shiftLeftPolicyResourceModel) *resource.ValidateConfigResponse {
	t.Helper()
	r := &shiftLeftPolicyResource{}
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	var resp resource.ValidateConfigResponse
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: configWith(t, schemaResp.Schema, model),
	}, &resp)
	return &resp
}

// tfsdk.Config has no Set, so seed a State and copy Raw (same tftypes shape).
func configWith(t *testing.T, sch fwschema.Schema, model shiftLeftPolicyResourceModel) tfsdk.Config {
	t.Helper()
	st := tfsdk.State{Schema: sch}
	if diags := st.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("failed to seed config: %v", diags)
	}
	return tfsdk.Config{Schema: sch, Raw: st.Raw}
}

func TestValidateConfig_AttachAllProjectsConflictsWithProjectsIds(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("iac"),
		Name:                     types.StringValue("iac"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		AttachAllProjects:        types.BoolValue(true),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("proj-1"),
		}),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for attach_all_projects + projects_ids")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "attach_all_projects") || !strings.Contains(detail, "projects_ids") {
		t.Fatalf("unexpected detail: %s", detail)
	}
}

func TestValidateConfig_AttachAllProjectsAloneIsAllowed(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("malicious_packages"),
		Name:                     types.StringValue("mp"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		AttachAllProjects:        types.BoolValue(true),
		ProjectsIds:              types.ListNull(types.StringType),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error for attach_all_projects alone: %v", resp.Diagnostics)
	}
}

// attach_all_projects = false is inert, so it must not collide with an explicit list.
func TestValidateConfig_AttachAllProjectsFalseAllowsProjectsIds(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("iac"),
		Name:                     types.StringValue("iac"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		AttachAllProjects:        types.BoolValue(false),
		ProjectsIds: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("proj-1"),
		}),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error for attach_all_projects=false + projects_ids: %v", resp.Diagnostics)
	}
}

func TestValidateConfig_ScmPostureRejectsAttachAllProjects(t *testing.T) {
	resp := validateConfig(t, shiftLeftPolicyResourceModel{
		Type:                     types.StringValue("scm_posture"),
		Name:                     types.StringValue("scm"),
		Disabled:                 types.BoolValue(false),
		WarnMode:                 types.BoolValue(false),
		PriorityFailureThreshold: types.StringValue("HIGH"),
		AttachAllProjects:        types.BoolValue(true),
		ProjectsIds:              types.ListNull(types.StringType),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for scm_posture + attach_all_projects")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "attach_all_projects") || !strings.Contains(detail, "scm_posture") {
		t.Fatalf("unexpected detail: %s", detail)
	}
}
