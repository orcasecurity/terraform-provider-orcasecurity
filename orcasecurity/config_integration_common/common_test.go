package config_integration_common

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buWithState embeds CommonFieldsWithBU so applyCommon can round-trip the
// business_units field through the State interface. A single variant attribute
// (echoed by the API) lets us assert Extract-style behavior isn't needed here.
type buState struct {
	CommonFieldsWithBU
}

// noBUState is the CommonFields flavour (no business_units).
type noBUState struct {
	CommonFields
}

// newSpec builds a minimal Spec[struct{}] with the given business-unit support and
// variant attributes. The CRUD refs are left nil because buildSchema/applyCommon/errorWrap
// never touch them.
func newSpec(supportsBUs bool, variant map[string]schema.Attribute) Spec[struct{}] {
	return Spec[struct{}]{
		TypeNameSuffix:        "_integration_test",
		UIName:                "Test integration",
		Description:           "desc",
		SupportsBusinessUnits: supportsBUs,
		VariantAttributes:     variant,
	}
}

// buildSchema must always emit the four cross-variant attributes with the documented
// requiredness/computed flags.
func TestBuildSchema_CommonAttributesAlwaysPresent(t *testing.T) {
	s := buildSchema(newSpec(false, nil))

	for _, name := range []string{"id", "template_name", "is_enabled", "is_default"} {
		if _, ok := s.Attributes[name]; !ok {
			t.Fatalf("expected common attribute %q to be present", name)
		}
	}

	if id, ok := s.Attributes["id"].(schema.StringAttribute); !ok || !id.Computed {
		t.Errorf("id must be a Computed string attribute, got %#v", s.Attributes["id"])
	}
	if tn, ok := s.Attributes["template_name"].(schema.StringAttribute); !ok || !tn.Required {
		t.Errorf("template_name must be a Required string attribute, got %#v", s.Attributes["template_name"])
	}
	if s.Description != "desc" {
		t.Errorf("schema description not carried through: %q", s.Description)
	}
}

// business_units must be emitted only when the Spec opts in via SupportsBusinessUnits.
func TestBuildSchema_BusinessUnitsGatedOnSupport(t *testing.T) {
	off := buildSchema(newSpec(false, nil))
	if _, ok := off.Attributes["business_units"]; ok {
		t.Error("business_units must be absent when SupportsBusinessUnits is false")
	}

	on := buildSchema(newSpec(true, nil))
	attr, ok := on.Attributes["business_units"]
	if !ok {
		t.Fatal("business_units must be present when SupportsBusinessUnits is true")
	}
	if set, ok := attr.(schema.SetAttribute); !ok || !set.Optional {
		t.Errorf("business_units must be an Optional set attribute, got %#v", attr)
	}
}

// Variant attributes must be merged into the schema alongside the common ones.
func TestBuildSchema_VariantAttributesMerged(t *testing.T) {
	variant := map[string]schema.Attribute{
		"api_token": schema.StringAttribute{Required: true, Sensitive: true},
		"host":      schema.StringAttribute{Required: true},
	}
	s := buildSchema(newSpec(false, variant))

	for name := range variant {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("expected variant attribute %q to be merged into schema", name)
		}
	}
	// Common attrs still present alongside the variant ones.
	if _, ok := s.Attributes["template_name"]; !ok {
		t.Error("merging variant attrs must not drop common attrs")
	}
}

// applyCommon must copy ID/IsEnabled/IsDefault from the API object onto the state's Common,
// and (when supported) populate business_units from the API list.
func TestApplyCommon_SetsCoreFieldsAndBusinessUnits(t *testing.T) {
	st := &buState{}
	// Seed a non-null planned set so BusinessUnitsFromAPI returns the API list rather than null.
	st.BusinessUnits, _ = types.SetValueFrom(context.Background(), types.StringType, []string{"seed"})

	api := APIObject{
		ID:            "abc-123",
		TemplateName:  "tn",
		IsEnabled:     true,
		IsDefault:     true,
		BusinessUnits: []string{"bu-1", "bu-2"},
	}
	var diags diag.Diagnostics
	applyCommon(context.Background(), st, api, true, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	c := st.GetCommon()
	if c.ID.ValueString() != "abc-123" {
		t.Errorf("ID not applied: %q", c.ID.ValueString())
	}
	if !c.IsEnabled.ValueBool() || !c.IsDefault.ValueBool() {
		t.Errorf("IsEnabled/IsDefault not applied: %v/%v", c.IsEnabled, c.IsDefault)
	}
	var got []string
	if d := c.BusinessUnits.ElementsAs(context.Background(), &got, false); d.HasError() {
		t.Fatalf("business_units decode: %v", d)
	}
	if len(got) != 2 || got[0] != "bu-1" {
		t.Errorf("business_units not applied from API: %v", got)
	}
}

// TemplateName must be overwritten only when the API returns a non-empty value; an empty
// API template_name must leave the pre-existing state value intact (create/update never echo
// it back, but Read does).
func TestApplyCommon_TemplateNameOnlyOverwrittenWhenNonEmpty(t *testing.T) {
	// Non-empty API value overwrites.
	st := &noBUState{}
	st.TemplateName = types.StringValue("original")
	var diags diag.Diagnostics
	applyCommon(context.Background(), st, APIObject{ID: "x", TemplateName: "from-api"}, false, &diags)
	if got := st.GetCommon().TemplateName.ValueString(); got != "from-api" {
		t.Errorf("non-empty API template_name should overwrite, got %q", got)
	}

	// Empty API value preserves the existing state value.
	st2 := &noBUState{}
	st2.TemplateName = types.StringValue("keep-me")
	applyCommon(context.Background(), st2, APIObject{ID: "x", TemplateName: ""}, false, &diags)
	if got := st2.GetCommon().TemplateName.ValueString(); got != "keep-me" {
		t.Errorf("empty API template_name must not clobber state, got %q", got)
	}
}

// When the variant does not support business_units, applyCommon must force the Common's
// BusinessUnits to a typed null regardless of what the API returned.
func TestApplyCommon_BusinessUnitsNullWhenUnsupported(t *testing.T) {
	st := &noBUState{}
	var diags diag.Diagnostics
	applyCommon(context.Background(), st, APIObject{ID: "x", BusinessUnits: []string{"ignored"}}, false, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	c := st.GetCommon()
	if !c.BusinessUnits.IsNull() {
		t.Errorf("business_units must be null when unsupported, got %v", c.BusinessUnits)
	}
	if c.BusinessUnits.ElementType(context.Background()) != types.StringType {
		t.Errorf("null business_units must retain string element type, got %v", c.BusinessUnits.ElementType(context.Background()))
	}
}

// errorWrap must produce gerund-form titles ("Error creating X") with the bare verb in the
// body ("Could not create X: ...") for every CRUD verb.
func TestErrorWrap_GerundTitlesForAllVerbs(t *testing.T) {
	cases := []struct {
		action    string
		wantTitle string
		wantBody  string
	}{
		{"create", "Error creating Widget", "Could not create Widget: boom"},
		{"read", "Error reading Widget", "Could not read Widget: boom"},
		{"update", "Error updating Widget", "Could not update Widget: boom"},
		{"delete", "Error deleting Widget", "Could not delete Widget: boom"},
	}
	for _, tc := range cases {
		t.Run(tc.action, func(t *testing.T) {
			var diags diag.Diagnostics
			errorWrap(&diags, tc.action, "Widget", errors.New("boom"))
			if len(diags) != 1 {
				t.Fatalf("expected exactly one diagnostic, got %d", len(diags))
			}
			d := diags[0]
			if d.Summary() != tc.wantTitle {
				t.Errorf("title = %q, want %q", d.Summary(), tc.wantTitle)
			}
			if d.Detail() != tc.wantBody {
				t.Errorf("body = %q, want %q", d.Detail(), tc.wantBody)
			}
		})
	}
}
