package custom_tag_rule_test

import (
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomTagRuleResource_JsonRule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_tag_rule" "test_json" {
  name      = "terraform-test-custom-tag-rule-json"
  rule_type = "json"
  rule = jsonencode({
    type   = "object_set"
    models = ["AwsEc2Instance"]
    with = {
      type     = "operation"
      operator = "and"
      values = [
        {
          key      = "IsInternetFacing"
          values   = [true]
          type     = "bool"
          operator = "eq"
        }
      ]
    }
  })
  tags = {
    "tf-test-exposure" = "internet-facing"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test_json", "name", "terraform-test-custom-tag-rule-json"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test_json", "rule_type", "json"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test_json", "tags.tf-test-exposure", "internet-facing"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_tag_rule.test_json", "id"),
				),
			},
			// refresh must not produce formatting-only diffs on the JSON rule
			{
				RefreshState: true,
			},
		},
	})
}

func TestAccCustomTagRuleResource_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_tag_rule" "test" {
  name        = "terraform-test-custom-tag-rule"
  description = "test description"
  rule        = "AwsEc2Instance"
  tags = {
    "tf-test-env" = "production"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "name", "terraform-test-custom-tag-rule"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "description", "test description"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "rule", "AwsEc2Instance"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "rule_type", "string"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "disabled", "false"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "tags.tf-test-env", "production"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_tag_rule.test", "id"),
				),
			},
			// import
			{
				ResourceName:      "orcasecurity_custom_tag_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + `
resource "orcasecurity_custom_tag_rule" "test" {
  name        = "terraform-test-custom-tag-rule-updated"
  description = "test description updated"
  rule        = "AwsEc2Instance with PublicIps"
  disabled    = true
  tags = {
    "tf-test-env"  = "staging"
    "tf-test-team" = "devops"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "name", "terraform-test-custom-tag-rule-updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "description", "test description updated"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "rule", "AwsEc2Instance with PublicIps"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "disabled", "true"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "tags.%", "2"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "tags.tf-test-env", "staging"),
					resource.TestCheckResourceAttr("orcasecurity_custom_tag_rule.test", "tags.tf-test-team", "devops"),
					resource.TestCheckResourceAttrSet("orcasecurity_custom_tag_rule.test", "id"),
				),
			},
		},
	})
}
