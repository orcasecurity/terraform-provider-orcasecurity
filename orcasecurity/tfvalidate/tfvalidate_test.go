package tfvalidate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var emailAttrTypes = map[string]attr.Type{
	"email":           types.ListType{ElemType: types.StringType},
	"asset_tag_keys":  types.ListType{ElemType: types.StringType},
	"custom_tag_keys": types.ListType{ElemType: types.StringType},
}

func stringList(values ...string) types.List {
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

func runAtLeastOneChildSet(value types.Object) validator.ObjectResponse {
	resp := validator.ObjectResponse{}
	AtLeastOneChildSet("email", "asset_tag_keys", "custom_tag_keys").ValidateObject(
		context.Background(),
		validator.ObjectRequest{
			Path:        path.Root("email_template"),
			ConfigValue: value,
		},
		&resp,
	)
	return resp
}

// Regression: a null object (e.g. config uses alert_dismissal_details, not
// email_template) must NOT be forced to supply email recipients.
func TestAtLeastOneChildSet_NullBlockNoError(t *testing.T) {
	resp := runAtLeastOneChildSet(types.ObjectNull(emailAttrTypes))
	if resp.Diagnostics.HasError() {
		t.Errorf("null block should not error, got: %v", resp.Diagnostics)
	}
}

func TestAtLeastOneChildSet_UnknownBlockNoError(t *testing.T) {
	resp := runAtLeastOneChildSet(types.ObjectUnknown(emailAttrTypes))
	if resp.Diagnostics.HasError() {
		t.Errorf("unknown block should not error, got: %v", resp.Diagnostics)
	}
}

func TestAtLeastOneChildSet_ConfiguredNoRecipientErrors(t *testing.T) {
	obj := types.ObjectValueMust(emailAttrTypes, map[string]attr.Value{
		"email":           types.ListNull(types.StringType),
		"asset_tag_keys":  types.ListNull(types.StringType),
		"custom_tag_keys": types.ListNull(types.StringType),
	})
	resp := runAtLeastOneChildSet(obj)
	if !resp.Diagnostics.HasError() {
		t.Errorf("configured block with no recipient should error")
	}
}

func TestAtLeastOneChildSet_OneRecipientNoError(t *testing.T) {
	obj := types.ObjectValueMust(emailAttrTypes, map[string]attr.Value{
		"email":           stringList("a@b.com"),
		"asset_tag_keys":  types.ListNull(types.StringType),
		"custom_tag_keys": types.ListNull(types.StringType),
	})
	resp := runAtLeastOneChildSet(obj)
	if resp.Diagnostics.HasError() {
		t.Errorf("block with a recipient should not error, got: %v", resp.Diagnostics)
	}
}

func TestAtLeastOneChildSet_UnknownChildDefersNoError(t *testing.T) {
	obj := types.ObjectValueMust(emailAttrTypes, map[string]attr.Value{
		"email":           types.ListUnknown(types.StringType),
		"asset_tag_keys":  types.ListNull(types.StringType),
		"custom_tag_keys": types.ListNull(types.StringType),
	})
	resp := runAtLeastOneChildSet(obj)
	if resp.Diagnostics.HasError() {
		t.Errorf("unknown child should defer validation, got: %v", resp.Diagnostics)
	}
}

// A control override that names only its id must be rejected at plan time: the
// posture policy API requires at least one of disabled/priority, and both are
// omitempty on the wire, so {"id": "..."} would otherwise reach the server and
// come back as a raw 400.
func TestAtLeastOneChildSet_PostureControlNeedsDisabledOrPriority(t *testing.T) {
	controlAttrTypes := map[string]attr.Type{
		"id":       types.StringType,
		"disabled": types.BoolType,
		"priority": types.StringType,
	}
	run := func(disabled, priority attr.Value) validator.ObjectResponse {
		resp := validator.ObjectResponse{}
		AtLeastOneChildSet("disabled", "priority").ValidateObject(
			context.Background(),
			validator.ObjectRequest{
				Path: path.Root("controls").AtListIndex(0),
				ConfigValue: types.ObjectValueMust(controlAttrTypes, map[string]attr.Value{
					"id":       types.StringValue("ctrl-1"),
					"disabled": disabled,
					"priority": priority,
				}),
			},
			&resp,
		)
		return resp
	}

	if resp := run(types.BoolNull(), types.StringNull()); !resp.Diagnostics.HasError() {
		t.Error("id-only control override must be rejected")
	}
	if resp := run(types.BoolValue(true), types.StringNull()); resp.Diagnostics.HasError() {
		t.Errorf("disabled-only control override must be accepted, got: %v", resp.Diagnostics)
	}
	if resp := run(types.BoolNull(), types.StringValue("HIGH")); resp.Diagnostics.HasError() {
		t.Errorf("priority-only control override must be accepted, got: %v", resp.Diagnostics)
	}
}
