package shift_left_policy

import (
	"context"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Older providers sent projects_ids in the scm_posture body; the API silently drops it, so those
// configs applied cleanly. It must keep planning (warning, no-op) rather than fail.
func TestValidateConfig_ScmPostureWarnsOnProjectsIds(t *testing.T) {
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
	if resp.Diagnostics.HasError() {
		t.Fatalf("scm_posture + projects_ids must not error: %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected one warning, got %v", resp.Diagnostics)
	}
	detail := resp.Diagnostics.Warnings()[0].Detail()
	if !strings.Contains(detail, "projects_ids") || !strings.Contains(detail, "scm_posture") {
		t.Fatalf("unexpected detail: %s", detail)
	}
}

func TestScmPostureProjectsIds_NoWriteAndKeptInState(t *testing.T) {
	configured := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("proj-1")})
	plan := &shiftLeftPolicyResourceModel{Type: types.StringValue("scm_posture"), ProjectsIds: configured}
	if projectsIdsChanged(plan, &shiftLeftPolicyResourceModel{Type: plan.Type, ProjectsIds: types.ListNull(types.StringType)}) {
		t.Error("scm_posture projects_ids must never trigger a projects write")
	}
	if err := (&shiftLeftPolicyResource{}).syncProjects(plan, "scm_posture", "p1"); err != nil {
		t.Errorf("syncProjects must be a no-op for scm_posture: %v", err)
	}
	got := apiToState(&api_client.ShiftLeftPolicy{ID: "p1", Type: "scm_posture"}, plan)
	if !got.ProjectsIds.Equal(configured) {
		t.Errorf("projects_ids = %v, want the configured %v (API never reports projects for scm_posture)", got.ProjectsIds, configured)
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
