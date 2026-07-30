package shift_left_integration

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AdoptLabels struct {
	NotFoundTitle  string
	NilReadTitle   string
	NilReadDetail  string
	ReadErrorTitle string
	DeleteLog      string
	MissingWarn    string // sprintf: unit id
}

func NewAdoptLabels(displayName string) AdoptLabels {
	lower := strings.ToLower(displayName)
	return AdoptLabels{
		NotFoundTitle:  displayName + " not found",
		NilReadTitle:   "Error reading " + lower + " after write",
		NilReadDetail:  "The " + lower + " was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		ReadErrorTitle: "Error reading " + displayName,
		DeleteLog:      "Deleting live " + lower + " (Terraform destroy tears down the integration).",
		MissingWarn:    displayName + " %s missing remotely",
	}
}

func ConfigureAPIClient(req resource.ConfigureRequest) *api_client.APIClient {
	if req.ProviderData == nil {
		return nil
	}
	return req.ProviderData.(*api_client.APIClient)
}

// ImportScopedInstallation: <installation_id>/<rest>; UUID rest sets id.
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

// ImportScopedUnit: UUID rest → id, else → nameAttr.
func ImportScopedUnit(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, nameAttr, expected string) {
	rest, resolved := ImportScopedInstallation(ctx, req, resp, expected)
	if resolved {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(nameAttr), rest)...)
}

// Disambiguates Orca unit UUIDs from SCM-side slugs/names on import.
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

func ReadUnit[T any](
	ctx context.Context,
	diags *diag.Diagnostics,
	labels AdoptLabels,
	unitID string,
	get func() (*T, error),
	remove func(context.Context),
) *T {
	unit, err := get()
	if err != nil {
		diags.AddError(labels.ReadErrorTitle, err.Error())
		return nil
	}
	if unit == nil {
		tflog.Warn(ctx, fmt.Sprintf(labels.MissingWarn, unitID))
		remove(ctx)
		return nil
	}
	return unit
}

func DeleteNoop(ctx context.Context, labels AdoptLabels) {
	tflog.Info(ctx, labels.DeleteLog)
}

// DeleteByLookup deletes a unit by id, resolving the id via lookup first when it's
// absent (state left by import may lack the Orca id). A nil lookup result means the
// unit is already gone remotely, which is treated as a successful delete.
func DeleteByLookup[T any](id string, lookup func() (*T, error), idOf func(*T) string, del func(string) error) error {
	if id == "" {
		found, err := lookup()
		if err != nil {
			return err
		}
		if found == nil {
			return nil
		}
		id = idOf(found)
	}
	return del(id)
}
