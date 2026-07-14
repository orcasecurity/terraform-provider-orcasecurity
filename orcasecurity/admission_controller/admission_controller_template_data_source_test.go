package admission_controller_test

import (
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
