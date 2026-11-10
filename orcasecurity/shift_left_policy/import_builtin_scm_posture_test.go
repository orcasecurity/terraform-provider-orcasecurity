package shift_left_policy_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The org-wide built-in scm_posture policy is owned exclusively by
// orcasecurity_shift_left_scm_posture_default_policy; importing it into the
// general policy resource would let two resource types write the same object.
func TestShiftLeftPolicyImport_BuiltinScmPostureRejected(t *testing.T) {
	const builtinID = "88888888-8888-8888-8888-888888888888"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": builtinID, "name": "Default SCM Posture Policy", "type": "scm_posture",
			"builtin": true, "disabled": false, "warn_mode": false,
			"priority_failure_threshold": "HIGH",
		})
	}))
	t.Cleanup(srv.Close)

	t.Setenv("TF_ACC", "1")
	t.Setenv("ORCASECURITY_API_ENDPOINT", srv.URL)
	t.Setenv("ORCASECURITY_API_TOKEN", "stub-token")

	config := orcasecurity.TestProviderConfig + `
resource "orcasecurity_shift_left_policy" "builtin_posture" {
  type                       = "scm_posture"
  name                       = "Default SCM Posture Policy"
  disabled                   = false
  warn_mode                  = false
  priority_failure_threshold = "HIGH"
  scm_posture {
  }
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:        config,
			ResourceName:  "orcasecurity_shift_left_policy.builtin_posture",
			ImportState:   true,
			ImportStateId: "scm_posture/" + builtinID,
			ExpectError:   regexp.MustCompile(`dedicated resource`),
		}},
	})
}
