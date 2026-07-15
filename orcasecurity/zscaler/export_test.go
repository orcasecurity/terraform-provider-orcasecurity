package zscaler

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// buildPayload / extract expose the variant's unexported Spec closures (via
// testutils.SpecFromResource) so the mapping tests read like the slack/monday ones.
func buildPayload(ctx context.Context, st cc.State, diags *diag.Diagnostics) api_client.ZscalerExternalServiceConfig {
	return testutils.SpecFromResource[api_client.ZscalerExternalServiceConfig](NewZscalerResource()).BuildPayload(ctx, st, diags)
}

func extract(o *api_client.ZscalerExternalServiceConfig, st cc.State, diags *diag.Diagnostics) cc.APIObject {
	return testutils.SpecFromResource[api_client.ZscalerExternalServiceConfig](NewZscalerResource()).Extract(o, st, diags)
}
