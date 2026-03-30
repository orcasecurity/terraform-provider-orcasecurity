package custom_compliance_framework_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	resourceAddr = "orcasecurity_custom_compliance_framework.test"

	attrID                       = "id"
	attrName                     = "name"
	attrDescription              = "description"
	attrSectionsCount            = "sections.#"
	attrSection0Name             = "sections.0.name"
	attrSection0TestsCount       = "sections.0.tests.#"
	attrSection0Test0RuleID      = "sections.0.tests.0.rule_id"
	attrSection0Test0FrameworkID = "sections.0.tests.0.rule_id_in_framework"
	attrSection0Test1RuleID      = "sections.0.tests.1.rule_id"
	attrSection0Test1FrameworkID = "sections.0.tests.1.rule_id_in_framework"
	attrSection1Name             = "sections.1.name"
	attrSection1TestsCount       = "sections.1.tests.#"
	attrSection1Test0RuleID      = "sections.1.tests.0.rule_id"
	attrSection1Test0FrameworkID = "sections.1.tests.0.rule_id_in_framework"

	testRuleID = "rc7bcf3b77f"
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Create Import Test"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Test create and import"),
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Section 1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0TestsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0FrameworkID, "1"),
					),
				},
				{
					ResourceName:            resourceAddr,
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Update Name Before"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Description stays the same"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Update Name After"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Description stays the same"),
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Section 1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0RuleID, testRuleID),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Original description"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Update Desc Test"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Updated description"),
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "1"),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Preserve ID After"),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Multi-Section Deep"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Deep field verification"),
						// Section counts
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "2"),
						// Section 0
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Access Control"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0TestsCount, "2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0FrameworkID, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test1RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test1FrameworkID, "2"),
						// Section 1
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Name, "Data Protection"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1TestsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Test0RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Test0FrameworkID, "3"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "1"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Section 1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Name, "Section 2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Test0RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Test0FrameworkID, "2"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "2"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Section A"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSection0TestsCount, "1"),
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
						resource.TestCheckResourceAttr(resourceAddr, attrSection0TestsCount, "2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0FrameworkID, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test1RuleID, testRuleID),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test1FrameworkID, "2"),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Full Update Before"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "Before full update"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "Old Section"),
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
						resource.TestCheckResourceAttrSet(resourceAddr, attrID),
						resource.TestCheckResourceAttr(resourceAddr, attrName, "TF Full Update After"),
						resource.TestCheckResourceAttr(resourceAddr, attrDescription, "After full update"),
						resource.TestCheckResourceAttr(resourceAddr, attrSectionsCount, "2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Name, "New Section"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0TestsCount, "2"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test0FrameworkID, "10"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection0Test1FrameworkID, "20"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Name, "Another New Section"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1TestsCount, "1"),
						resource.TestCheckResourceAttr(resourceAddr, attrSection1Test0FrameworkID, "30"),
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
