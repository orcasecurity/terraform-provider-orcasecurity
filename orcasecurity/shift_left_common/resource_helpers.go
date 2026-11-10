package shift_left_common

import (
	"context"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func ConfigureAPIClient(req resource.ConfigureRequest) *api_client.APIClient {
	if req.ProviderData == nil {
		return nil
	}
	return req.ProviderData.(*api_client.APIClient)
}

// Import ID: <installation_id>/<rest>; UUID rest sets id.
func ImportScopedInstallation(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, expected string) (rest string, resolved bool) {
	installationID, rest, ok := strings.Cut(req.ID, "/")
	if !ok || installationID == "" || rest == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected "+expected)
		return "", true
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), installationID)...)
	if LooksLikeUUID(rest) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rest)...)
		return "", true
	}
	return rest, false
}

// Non-UUID rest goes to nameAttr.
func ImportScopedUnit(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, nameAttr, expected string) {
	rest, resolved := ImportScopedInstallation(ctx, req, resp, expected)
	if resolved {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(nameAttr), rest)...)
}

func LooksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
	}
	return true
}
