package alert_common

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/createrecovery"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Create runs op, then sets the new rule's remediation text, so a recovered rule gets it too.
func Create[T any](ctx context.Context, client *api_client.APIClient, op api_client.CreateOp[T], ruleType func(*T) string,
	text *api_client.CustomSonarAlertRemediationText, diags *diag.Diagnostics) (*T, error) {
	created, err := createrecovery.Run(ctx, op, diags)
	if err != nil {
		return nil, err
	}
	if err := client.SetCreatedCustomAlertRemediation(op.ID(created), ruleType(created), text); err != nil {
		return nil, err
	}
	return created, nil
}
