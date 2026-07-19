package servicenow_sir_template

import (
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"
)

// The SIR template resource must advertise its own type-name suffix (distinct from ITSM even
// though both share the sn_incidents service), carry the ServiceNow-specific mapping attributes
// plus the shared template_name key, hide business_units, and support import by template_name.
func TestServiceNowSIRTemplateResource_Contract(t *testing.T) {
	testutils.CheckTemplateResource(t, testutils.TemplateResourceSpec{
		NewResource: NewServiceNowSIRTemplateResource,
		TypeName:    "orcasecurity_integration_servicenow_sir_template",
		Attrs:       []string{"template_name", "mapping_json", "resolution_status", "resource_id"},
		Forbidden:   []string{"business_units"},
	})
}
