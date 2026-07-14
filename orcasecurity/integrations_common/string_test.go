package integrations_common

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestOptionalStringMatchPlan(t *testing.T) {
	empty := ""
	value := "desc"

	t.Run("nil API value with null prior stays null", func(t *testing.T) {
		if got := OptionalStringMatchPlan(types.StringNull(), nil); !got.IsNull() {
			t.Errorf("expected null, got %v", got)
		}
	})

	t.Run("empty API value with null prior stays null", func(t *testing.T) {
		if got := OptionalStringMatchPlan(types.StringNull(), &empty); !got.IsNull() {
			t.Errorf("expected null, got %v", got)
		}
	})

	t.Run("explicitly configured empty string is preserved", func(t *testing.T) {
		got := OptionalStringMatchPlan(types.StringValue(""), &empty)
		if got.IsNull() || got.ValueString() != "" {
			t.Errorf("expected empty string, got %v", got)
		}
	})

	t.Run("API value wins over prior", func(t *testing.T) {
		got := OptionalStringMatchPlan(types.StringValue("old"), &value)
		if got.ValueString() != "desc" {
			t.Errorf("expected desc, got %v", got)
		}
	})

	t.Run("nil API value with non-empty prior goes null", func(t *testing.T) {
		// The remote value was cleared: state must not resurrect the prior.
		if got := OptionalStringMatchPlan(types.StringValue("old"), nil); !got.IsNull() {
			t.Errorf("expected null, got %v", got)
		}
	})
}
