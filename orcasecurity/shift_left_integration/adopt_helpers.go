package shift_left_integration

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
// absent (state left by import may lack the Orca id). When id is empty and lookup
// returns nil, it returns ErrUnitNotFound — DoDelete treats that as success (already
// gone), but create-rollback must warn: Integrate just succeeded, so a nil lookup is
// the silent-orphan case, not a clean delete.
func DeleteByLookup[T any](id string, lookup func() (*T, error), idOf func(*T) string, del func(string) error) error {
	if id == "" {
		found, err := lookup()
		if err != nil {
			return err
		}
		if found == nil {
			return ErrUnitNotFound
		}
		id = idOf(found)
	}
	return del(id)
}
