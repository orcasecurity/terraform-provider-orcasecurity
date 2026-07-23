package shift_left_integration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// AdoptLabels carries provider-specific error copy for the shared adopt write path.
type AdoptLabels struct {
	NotFoundTitle  string
	NilReadTitle   string
	NilReadDetail  string
	ReadErrorTitle string
	DeleteLog      string
	MissingWarn    string // sprintf: unit id
}

// NewAdoptLabels derives the standard adopt-resource error/log copy from a
// unit display name (e.g. "GitHub installation", "Azure DevOps account"), so
// per-provider packages declare one string instead of six formulaic messages.
func NewAdoptLabels(displayName string) AdoptLabels {
	lower := strings.ToLower(displayName)
	return AdoptLabels{
		NotFoundTitle:  displayName + " not found",
		NilReadTitle:   "Error reading " + lower + " after write",
		NilReadDetail:  "The " + lower + " was configured but could not be read back; the API may not have propagated the change yet. Re-run terraform apply.",
		ReadErrorTitle: "Error reading " + displayName,
		DeleteLog:      "Removing " + displayName + " from state; the live integration is left untouched.",
		MissingWarn:    displayName + " %s missing remotely",
	}
}

// ConfigureAPIClient extracts the API client from provider data.
func ConfigureAPIClient(req resource.ConfigureRequest) *api_client.APIClient {
	if req.ProviderData == nil {
		return nil
	}
	return req.ProviderData.(*api_client.APIClient)
}

// ImportSlashPair parses "<left>/<right>" import IDs into two state attributes.
func ImportSlashPair(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, leftAttr, rightAttr, expected string) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected "+expected)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(leftAttr), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(rightAttr), parts[1])...)
}

// AdoptWrite runs WriteAdopted and maps errors into diagnostics. Returns nil
// when diagnostics were added.
func AdoptWrite[T any](diags *diag.Diagnostics, req AdoptWriteRequest[T]) *T {
	unit, err := WriteAdopted(req)
	if errors.Is(err, ErrUnitNotFound) {
		diags.AddError(req.Labels.NotFoundTitle, req.NotFoundMsg)
		return nil
	}
	if err != nil {
		diags.AddError(req.WriteErrorTitle, err.Error())
		return nil
	}
	if unit == nil {
		diags.AddError(req.Labels.NilReadTitle, req.Labels.NilReadDetail)
		return nil
	}
	return unit
}

// ReadUnit loads a unit for Read; removes the resource when missing remotely.
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

// DeleteNoop logs that the live SCM integration is left untouched.
func DeleteNoop(ctx context.Context, labels AdoptLabels) {
	tflog.Info(ctx, labels.DeleteLog)
}
