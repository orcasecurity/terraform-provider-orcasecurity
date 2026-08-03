package shift_left_unit

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The GitLab group PUT goes through UpdateConfigurationSettingsRequest, which
// types skip_check_runs as the three-value PerformActionStatus. This resource
// previously declared the two-value GitLab *repository* enum, rejecting at plan
// time a value the API accepts (verified live against the group endpoint).
func TestResourceSchema_SkipCheckRunsAcceptsOnlyOnInternalIssue(t *testing.T) {
	cfg, ok := gitlabGroupSchema().Attributes["configuration_settings"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatal("configuration_settings must be a SingleNestedAttribute")
	}
	attr, ok := cfg.Attributes["skip_check_runs"].(rschema.StringAttribute)
	if !ok {
		t.Fatal("skip_check_runs must be a StringAttribute")
	}

	for _, value := range []string{"ALWAYS", "NEVER", "ONLY_ON_INTERNAL_ISSUE"} {
		for _, v := range attr.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("configuration_settings").AtName("skip_check_runs"),
				ConfigValue: types.StringValue(value),
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("GitLab group skip_check_runs must accept %q: %v", value, resp.Diagnostics)
			}
		}
	}
}
