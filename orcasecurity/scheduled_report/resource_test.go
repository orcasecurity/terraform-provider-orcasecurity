package scheduled_report_test

import (
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ResourceType = "orcasecurity_scheduled_report"
	Resource     = "terraformTestResource"
	OrcaObject   = "terraformTestResourceInOrca"
)

func TestScheduledReportResource_Basic(t *testing.T) {
	resourceAddress := fmt.Sprintf("%s.%s", ResourceType, Resource)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  name              = "%s"
  type              = "alerts_svl"
  format            = "csv"
  recurrence        = "daily"
  first_report_date = "2099-01-01T00:00:00Z"
  export_time       = "00:00:00"

  recipients_emails    = ["test@orca.security"]
  custom_email_subject = "custom subject"
  custom_email_content = "some custom content"

  # Every optional attribute here is dropped in the update step below. Because
  # updates are a PATCH, an attribute that is not sent stays live on the server,
  # so a removal that does not reach the API shows up as a non-empty refresh plan
  # after step 3.
  #
  # compression is why this step uses csv: the API only allows compression_type
  # on the text formats, and rejects it for pdf, xlsx and html.
  columns     = ["OrcaScore", "Title"]
  s3_path     = "terraform-test/reports"
  compression = ".zip"

  sonar_query_params = jsonencode({ max_tier = 5 })
  query_filters      = jsonencode({ show_informational_alerts = true })

  sonar_query = jsonencode({
    models = ["Alert"]
    type   = "object_set"
    with = {
      operator = "and"
      type     = "operation"
      values = [
        {
          key      = "RiskLevel"
          values   = ["critical", "high"]
          type     = "str"
          operator = "in"
        }
      ]
    }
  })
}
`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceAddress, "id"),
					resource.TestCheckResourceAttr(resourceAddress, "name", OrcaObject),
					resource.TestCheckResourceAttr(resourceAddress, "type", "alerts_svl"),
					resource.TestCheckResourceAttr(resourceAddress, "format", "csv"),
					resource.TestCheckResourceAttr(resourceAddress, "recurrence", "daily"),
					resource.TestCheckResourceAttr(resourceAddress, "status", "active"),
					resource.TestCheckResourceAttr(resourceAddress, "recipients_emails.0", "test@orca.security"),
					resource.TestCheckResourceAttr(resourceAddress, "share_to_slack", "false"),
					resource.TestCheckResourceAttr(resourceAddress, "custom_email_subject", "custom subject"),
					resource.TestCheckResourceAttr(resourceAddress, "s3_path", "terraform-test/reports"),
					resource.TestCheckResourceAttr(resourceAddress, "compression", ".zip"),
					resource.TestCheckResourceAttr(resourceAddress, "columns.#", "2"),
				),
			},
			// import
			{
				ResourceName:      resourceAddress,
				ImportState:       true,
				ImportStateVerify: true,
				// JSON-encoded attributes may be normalized by the API,
				// so their imported representation can differ from the config.
				ImportStateVerifyIgnore: []string{"sonar_query"},
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
  name              = "%s updated"
  type              = "alerts_svl"
  format            = "pdf"
  recurrence        = "weekly"
  first_report_date = "2099-01-01T00:00:00Z"
  export_time       = "12:00:00"
  status            = "disabled"

  recipients_emails = ["test@orca.security"]

  sonar_query = jsonencode({
    models = ["Alert"]
    type   = "object_set"
    with = {
      operator = "and"
      type     = "operation"
      values = [
        {
          key      = "RiskLevel"
          values   = ["critical", "high", "medium"]
          type     = "str"
          operator = "in"
        }
      ]
    }
  })
}
`, ResourceType, Resource, OrcaObject),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceAddress, "name", OrcaObject+" updated"),
					resource.TestCheckResourceAttr(resourceAddress, "format", "pdf"),
					resource.TestCheckResourceAttr(resourceAddress, "recurrence", "weekly"),
					resource.TestCheckResourceAttr(resourceAddress, "status", "disabled"),
					resource.TestCheckResourceAttr(resourceAddress, "export_time", "12:00:00"),
					// Dropped from the config above: each must now be absent from
					// state rather than retaining the value step 1 set.
					resource.TestCheckNoResourceAttr(resourceAddress, "custom_email_subject"),
					resource.TestCheckNoResourceAttr(resourceAddress, "custom_email_content"),
					resource.TestCheckNoResourceAttr(resourceAddress, "s3_path"),
					resource.TestCheckNoResourceAttr(resourceAddress, "compression"),
					resource.TestCheckNoResourceAttr(resourceAddress, "sonar_query_params"),
					resource.TestCheckNoResourceAttr(resourceAddress, "query_filters"),
				),
			},
		},
	})
}
