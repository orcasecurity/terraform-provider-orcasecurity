package automation_v2

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func TestSetOptionalString_SkipNullUnknownEmpty(t *testing.T) {
	payload := map[string]interface{}{}

	setOptionalString(payload, "null", types.StringNull())
	setOptionalString(payload, "unknown", types.StringUnknown())
	setOptionalString(payload, "empty", types.StringValue(""))
	setOptionalString(payload, "value", types.StringValue("hello"))

	if _, ok := payload["null"]; ok {
		t.Errorf("null value should be skipped")
	}
	if _, ok := payload["unknown"]; ok {
		t.Errorf("unknown value should be skipped")
	}
	if _, ok := payload["empty"]; ok {
		t.Errorf("empty string should be skipped")
	}
	if got := payload["value"]; got != "hello" {
		t.Errorf("expected hello, got %v", got)
	}
}

func TestAppendExternalConfigAction_NilTemplate(t *testing.T) {
	out := appendExternalConfigAction(nil, nil, api_client.AutomationSlackID)
	if len(out) != 0 {
		t.Errorf("expected no action for nil template, got %d", len(out))
	}
}

func TestAppendExternalConfigAction_Populated(t *testing.T) {
	tmpl := &automationV2ExternalConfigTemplateModel{
		ExternalConfigID: types.StringValue("cfg-123"),
	}

	out := appendExternalConfigAction(nil, tmpl, api_client.AutomationSlackID)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	if out[0].Type != api_client.AutomationSlackID {
		t.Errorf("expected Type %d, got %d", api_client.AutomationSlackID, out[0].Type)
	}
	if out[0].ExternalConfig == nil || *out[0].ExternalConfig != "cfg-123" {
		t.Errorf("expected ExternalConfig cfg-123, got %v", out[0].ExternalConfig)
	}
	if len(out[0].Data) != 0 {
		t.Errorf("expected empty Data, got %v", out[0].Data)
	}
}

func TestAppendExternalConfigWithParentAction_OmitParentWhenNull(t *testing.T) {
	tmpl := &automationV2ExternalConfigWithParentTemplateModel{
		ExternalConfigID: types.StringValue("cfg-123"),
		ParentIssueID:    types.StringNull(),
	}

	out := appendExternalConfigWithParentAction(nil, tmpl, api_client.AutomationJiraID)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	if _, ok := out[0].Data["parent_id"]; ok {
		t.Errorf("expected parent_id absent when null")
	}
}

func TestAppendExternalConfigWithParentAction_IncludeParent(t *testing.T) {
	tmpl := &automationV2ExternalConfigWithParentTemplateModel{
		ExternalConfigID: types.StringValue("cfg-123"),
		ParentIssueID:    types.StringValue("PROJ-1"),
	}

	out := appendExternalConfigWithParentAction(nil, tmpl, api_client.AutomationJiraID)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	if got := out[0].Data["parent_id"]; got != "PROJ-1" {
		t.Errorf("expected parent_id PROJ-1, got %v", got)
	}
}

func TestGenerateV2Actions_ApiTokenTemplate(t *testing.T) {
	plan := &automationV2ResourceModel{
		ApiTokenTemplate: &automationV2ExternalConfigTemplateModel{
			ExternalConfigID: types.StringValue("09827e5e-19d2-41dd-87b1-8f90009773a6"),
		},
	}

	actions, err := generateV2Actions(plan, nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	a := actions[0]
	if a.Type != api_client.AutomationSiemID {
		t.Errorf("expected Type %d, got %d", api_client.AutomationSiemID, a.Type)
	}
	if a.SiemToken == nil || *a.SiemToken != "09827e5e-19d2-41dd-87b1-8f90009773a6" {
		t.Errorf("expected SiemToken 09827e5e-19d2-41dd-87b1-8f90009773a6, got %v", a.SiemToken)
	}
	if a.ExternalConfig != nil {
		t.Errorf("expected nil ExternalConfig, got %v", *a.ExternalConfig)
	}
	if len(a.Data) != 0 {
		t.Errorf("expected empty Data, got %v", a.Data)
	}
}

func TestAppendReasonJustificationAction_EmptyValuesSkipped(t *testing.T) {
	out := appendReasonJustificationAction(nil, api_client.AutomationAlertDismissalID,
		types.StringValue(""), types.StringNull(), nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	if _, ok := out[0].Data["reason"]; ok {
		t.Errorf("expected reason absent when empty")
	}
	if _, ok := out[0].Data["justification"]; ok {
		t.Errorf("expected justification absent when null")
	}
}

func TestAppendReasonJustificationAction_PopulatedWithExtra(t *testing.T) {
	extra := map[string]interface{}{"days": int64(7)}
	out := appendReasonJustificationAction(nil, api_client.AutomationSnoozeID,
		types.StringValue("vacation"), types.StringValue("annual leave"), extra)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	want := map[string]interface{}{
		"days":          int64(7),
		"reason":        "vacation",
		"justification": "annual leave",
	}
	if !reflect.DeepEqual(out[0].Data, want) {
		t.Errorf("expected %v, got %v", want, out[0].Data)
	}
}
