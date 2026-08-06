package tfvalidate

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Like objectvalidator.AtLeastOneOf, but skips when the nested object itself is null.
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
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()
	for _, name := range v.attributes {
		val, ok := attrs[name]
		if !ok {
			continue
		}
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
