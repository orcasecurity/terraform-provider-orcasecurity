package testutils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
)

// RequireLinkedCollectionAttribute asserts the schema contract for a collection
// whose values another owner can also write — the Orca UI, or a second resource.
// Optional so a config can own it, Computed so a config that stays silent does
// not plan the remote values away, and UseStateForUnknown so an update does not
// replan them. Optional alone means Terraform deletes what it did not ask for.
func RequireLinkedCollectionAttribute(t *testing.T, attribute schema.ListNestedAttribute) {
	t.Helper()
	if !attribute.Optional {
		t.Error("attribute must stay Optional so a config can own the values")
	}
	if !attribute.Computed {
		t.Error("attribute must be Computed so a silent config keeps the remote values")
	}
	for _, modifier := range attribute.PlanModifiers {
		if modifier == listplanmodifier.UseStateForUnknown() {
			return
		}
	}
	t.Error("attribute needs UseStateForUnknown so updates do not replan the values")
}
