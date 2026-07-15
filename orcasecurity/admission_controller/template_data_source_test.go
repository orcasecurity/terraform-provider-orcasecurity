package admission_controller_test

import (
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAdmissionControllerTemplateDataSource_ByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "test" {
  name = "k8sallowedrepos"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.orcasecurity_admission_controller_template.test", "id"),
					resource.TestCheckResourceAttr("data.orcasecurity_admission_controller_template.test", "display_name", "Allowed Container Registries"),
					resource.TestCheckResourceAttr("data.orcasecurity_admission_controller_template.test", "controller_type", "gatekeeper"),
					resource.TestCheckResourceAttr("data.orcasecurity_admission_controller_template.test", "supported_kinds.0", "Pod"),
				),
			},
		},
	})
}

func TestAccAdmissionControllerTemplateDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "missing" {
  name = "tf-acc-no-such-template"
}
`,
				ExpectError: regexp.MustCompile(`Admission controller template not found`),
			},
		},
	})
}

// The two halves of the ExactlyOneOf(name, display_name) config validator:
// neither and both must fail at plan time.
func TestAccAdmissionControllerTemplateDataSource_NoSelector(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "invalid" {
}
`,
				ExpectError: regexp.MustCompile(`(?s)Exactly one of.*name.*display_name`),
			},
		},
	})
}

func TestAccAdmissionControllerTemplateDataSource_BothSelectors(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "invalid" {
  name         = "k8sallowedrepos"
  display_name = "Allowed Container Registries"
}
`,
				ExpectError: regexp.MustCompile(`(?s)Exactly one of.*name.*display_name`),
			},
		},
	})
}

func TestAccAdmissionControllerTemplateDataSource_ByDisplayName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + `
data "orcasecurity_admission_controller_template" "test" {
  display_name = "Allowed Container Registries"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.orcasecurity_admission_controller_template.test", "name", "k8sallowedrepos"),
				),
			},
		},
	})
}
