package servicenow_itsm_template

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// The ITSM template resource must advertise its own type-name suffix (so it does not collide with
// the SIR variant), carry the ServiceNow-specific mapping attributes plus the shared
// template_name key, hide business_units, and support import by template_name.
func TestServiceNowITSMTemplateResource_Contract(t *testing.T) {
	testutils.CheckTemplateResource(t, testutils.TemplateResourceSpec{
		NewResource: NewServiceNowITSMTemplateResource,
		TypeName:    "orcasecurity_integration_servicenow_itsm_template",
		Attrs:       []string{"template_name", "mapping_json", "resolution_status", "resource_id"},
		Forbidden:   []string{"business_units"},
	})
}
