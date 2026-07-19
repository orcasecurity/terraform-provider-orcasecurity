package automation_v2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// Regression: reason/justification on alert_dismissal_details must be plain
// Optional (non-Computed) string attributes with NO plan modifiers. A plan
// modifier normalizing "" -> null here makes the planned value differ from
// config, which the Plugin Framework rejects with "planned value ... does not
// match config value" for an Optional (non-Computed) attribute. See the
// optionalStringAttr comment.
func TestAlertDismissalReasonHasNoPlanModifier(t *testing.T) {
	r := &automationV2Resource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	block, ok := resp.Schema.Attributes["alert_dismissal_details"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("alert_dismissal_details is not a SingleNestedAttribute")
	}

	for _, name := range []string{"reason", "justification"} {
		attr, ok := block.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a StringAttribute", name)
		}
		if !attr.Optional {
			t.Errorf("%s must be Optional", name)
		}
		if attr.Computed {
			t.Errorf("%s must not be Computed", name)
		}
		if len(attr.PlanModifiers) != 0 {
			t.Errorf("%s must have no plan modifiers, got %d", name, len(attr.PlanModifiers))
		}
	}
}

// priority must be plain Optional (non-Computed) with NO plan modifiers:
// Computed would make every external reorder show as "changed outside
// Terraform" for users who never set it, and a plan modifier on an Optional
// non-Computed attribute is rejected by the framework (see
// optionalStringAttr comment). Exactly one validator (AtLeast(1)) enforces
// the backend's min_value=1.
func TestPrioritySchemaIsOptionalNonComputed(t *testing.T) {
	r := &automationV2Resource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["priority"].(schema.Int64Attribute)
	if !ok {
		t.Fatalf("priority is not an Int64Attribute")
	}
	if !attr.Optional {
		t.Error("priority must be Optional")
	}
	if attr.Computed {
		t.Error("priority must not be Computed")
	}
	if len(attr.PlanModifiers) != 0 {
		t.Errorf("priority must have no plan modifiers, got %d", len(attr.PlanModifiers))
	}
	if len(attr.Validators) != 1 {
		t.Errorf("priority must have exactly 1 validator (AtLeast(1)), got %d", len(attr.Validators))
	}
}
