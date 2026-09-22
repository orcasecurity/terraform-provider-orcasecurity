package custom_discovery_alert

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The links are also written by the Orca UI and by the custom compliance
// framework resource. Conversion is shared and covered in alert_common; what is
// resource-specific is that this schema declares the shared contract.
func TestComplianceFrameworksIsALinkedCollection(t *testing.T) {
	schemaResp := &resource.SchemaResponse{}
	(&customDiscoveryAlertResource{}).Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	attribute, ok := schemaResp.Schema.Attributes["compliance_frameworks"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("compliance_frameworks must be a ListNestedAttribute")
	}
	testutils.RequireLinkedCollectionAttribute(t, attribute)
}
