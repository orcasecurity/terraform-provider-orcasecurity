package automation_v2_priority_order_test

import (
	"fmt"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckServerPriority asserts the automation behind resourceName holds
// the given priority on the server, reading through the live API rather than
// trusting Terraform state.
func testAccCheckServerPriority(resourceName string, want int64) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}
		endpoint := os.Getenv("ORCASECURITY_API_ENDPOINT")
		token := os.Getenv("ORCASECURITY_API_TOKEN")
		client, err := api_client.NewAPIClient(&endpoint, &token)
		if err != nil {
			return err
		}
		instance, err := client.GetAutomationV2(rs.Primary.ID)
		if err != nil {
			return err
		}
		if instance == nil || instance.Priority == nil {
			return fmt.Errorf("automation %s has no priority on the server", rs.Primary.ID)
		}
		if *instance.Priority != want {
			return fmt.Errorf("automation %s: want priority %d, got %d", rs.Primary.ID, want, *instance.Priority)
		}
		return nil
	}
}

// priorityOrderConfig provisions two automations and a priority_order resource
// listing them in the given order. Referencing the automations' ids creates the
// implicit dependency that makes Terraform build them before ordering and tear
// everything down at the end.
func priorityOrderConfig(first, second string) string {
	return orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_automation_v2" "a" {
  name        = "tf-acc-order-a"
  description = "tf-acc-order-a"
  status      = "enabled"
  filter = {
    sonar_query = jsonencode({ models = ["Alert"], type = "object_set" })
  }
  alert_dismissal_details = { reason = "acceptance test" }
}

resource "orcasecurity_automation_v2" "b" {
  name        = "tf-acc-order-b"
  description = "tf-acc-order-b"
  status      = "enabled"
  filter = {
    sonar_query = jsonencode({ models = ["Alert"], type = "object_set" })
  }
  alert_dismissal_details = { reason = "acceptance test" }
}

resource "orcasecurity_automation_v2_priority_order" "test" {
  automation_ids = [%s, %s]
}
`, first, second)
}

// TestAccAutomationV2PriorityOrderResource confirms the bulk ordering resource
// against the live API: it drives two automations to the top of the global
// evaluation order, then reverses them, verifying the server-side priority each
// time.
func TestAccAutomationV2PriorityOrderResource(t *testing.T) {
	const (
		aID = "orcasecurity_automation_v2.a.id"
		bID = "orcasecurity_automation_v2.b.id"
	)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { orcasecurity.TestAccPreCheck(t) },
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Order [A, B]: A gets priority 1, B gets priority 2.
			{
				Config: priorityOrderConfig(aID, bID),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerPriority("orcasecurity_automation_v2.a", 1),
					testAccCheckServerPriority("orcasecurity_automation_v2.b", 2),
					resource.TestCheckResourceAttr("orcasecurity_automation_v2_priority_order.test", "automation_ids.#", "2"),
					resource.TestCheckResourceAttrPair(
						"orcasecurity_automation_v2_priority_order.test", "automation_ids.0",
						"orcasecurity_automation_v2.a", "id"),
				),
			},
			// Reorder [B, A]: B gets priority 1, A gets priority 2.
			{
				Config: priorityOrderConfig(bID, aID),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckServerPriority("orcasecurity_automation_v2.b", 1),
					testAccCheckServerPriority("orcasecurity_automation_v2.a", 2),
					resource.TestCheckResourceAttrPair(
						"orcasecurity_automation_v2_priority_order.test", "automation_ids.0",
						"orcasecurity_automation_v2.b", "id"),
				),
			},
		},
	})
}
