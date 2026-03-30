package custom_compliance_framework_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomComplianceFrameworkResource(t *testing.T) {
	tests := map[string]struct {
		steps []resource.TestStep
	}{
		"create and import": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Create Import Test"
    description = "Test create and import"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Create Import Test"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Test create and import"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Section 1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id_in_framework", "1"),
					),
				},
				{
					ResourceName:            "orcasecurity_custom_compliance_framework.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: []string{"sections"},
				},
			},
		},
		"update name only": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Update Name Before"
    description = "Description stays the same"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Update Name Before"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Description stays the same"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Update Name After"
    description = "Description stays the same"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Update Name After"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Description stays the same"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Section 1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id", "rc7bcf3b77f"),
					),
				},
			},
		},
		"update description only": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Update Desc Test"
    description = "Original description"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Original description"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Update Desc Test"
    description = "Updated description"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Update Desc Test"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Updated description"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "1"),
					),
				},
			},
		},
		"update preserves id": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Preserve ID Before"
    description = "Check ID stability"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Preserve ID After"
    description = "Check ID stability updated"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						// ID should still be set (not empty) and Terraform should not have recreated the resource
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Preserve ID After"),
					),
				},
			},
		},
		"multiple sections with deep field checks": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Multi-Section Deep"
    description = "Deep field verification"
    sections = [
        {
            name = "Access Control"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                },
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "2"
                }
            ]
        },
        {
            name = "Data Protection"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "3"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Multi-Section Deep"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Deep field verification"),
						// Section counts
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "2"),
						// Section 0
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Access Control"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.#", "2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id_in_framework", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.1.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.1.rule_id_in_framework", "2"),
						// Section 1
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.name", "Data Protection"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.0.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.0.rule_id_in_framework", "3"),
					),
				},
			},
		},
		"add section on update": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Add Section"
    description = "Start with one section"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "1"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Add Section"
    description = "Start with one section"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        },
        {
            name = "Section 2"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "2"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Section 1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.name", "Section 2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.0.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.0.rule_id_in_framework", "2"),
					),
				},
			},
		},
		"remove section on update": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Remove Section"
    description = "Start with two sections"
    sections = [
        {
            name = "Section A"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        },
        {
            name = "Section B"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "2"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "2"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Remove Section"
    description = "Start with two sections"
    sections = [
        {
            name = "Section A"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Section A"),
					),
				},
			},
		},
		"add test to existing section": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Add Test"
    description = "Start with one test"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.#", "1"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Add Test"
    description = "Start with one test"
    sections = [
        {
            name = "Section 1"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                },
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "2"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.#", "2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id_in_framework", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.1.rule_id", "rc7bcf3b77f"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.1.rule_id_in_framework", "2"),
					),
				},
			},
		},
		"update all fields at once": {
			steps: []resource.TestStep{
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Full Update Before"
    description = "Before full update"
    sections = [
        {
            name = "Old Section"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "1"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Full Update Before"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "Before full update"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "Old Section"),
					),
				},
				{
					Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_compliance_framework" "test" {
    name        = "TF Full Update After"
    description = "After full update"
    sections = [
        {
            name = "New Section"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "10"
                },
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "20"
                }
            ]
        },
        {
            name = "Another New Section"
            tests = [
                {
                    rule_id              = "rc7bcf3b77f"
                    rule_id_in_framework = "30"
                }
            ]
        }
    ]
}
`,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("orcasecurity_custom_compliance_framework.test", "id"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "name", "TF Full Update After"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "description", "After full update"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.#", "2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.name", "New Section"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.#", "2"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.0.rule_id_in_framework", "10"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.0.tests.1.rule_id_in_framework", "20"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.name", "Another New Section"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.#", "1"),
						resource.TestCheckResourceAttr("orcasecurity_custom_compliance_framework.test", "sections.1.tests.0.rule_id_in_framework", "30"),
					),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
				Steps:                    tc.steps,
			})
		})
	}
}
