package automation_v2

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// atLeastOneChildSet validates that, when a nested object block is configured
// (non-null), at least one of the named child attributes is set. Unlike
// objectvalidator.AtLeastOneOf, it does NOT fire when the object itself is null,
// so the block stays optional.
type atLeastOneChildSet struct {
	attributes []string
}

func AtLeastOneChildSet(attributes ...string) validator.Object {
	return atLeastOneChildSet{attributes: attributes}
}

func (v atLeastOneChildSet) Description(_ context.Context) string {
	return fmt.Sprintf("when set, at least one of these attributes must be specified: %s", strings.Join(v.attributes, ", "))
}

func (v atLeastOneChildSet) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v atLeastOneChildSet) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	// Block not configured, or not yet known - nothing to enforce.
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	for _, name := range v.attributes {
		val, ok := attrs[name]
		if !ok {
			continue
		}
		// Unknown value may resolve to something set - don't error early.
		if val.IsUnknown() {
			return
		}
		if !val.IsNull() {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Missing required attribute",
		fmt.Sprintf("At least one of %s must be specified.", strings.Join(v.attributes, ", ")),
	)
}
