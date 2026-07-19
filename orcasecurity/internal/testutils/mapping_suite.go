package testutils

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// suiteBUs is the canonical business_units fixture RunMappingSuite feeds BU-capable variants.
var suiteBUs = []string{"bu-1", "bu-2"}

// StateCheck pairs a planned state with an assertion for RunMappingSuite's zero-config extract
// beats: an empty API config must keep the planned values (no spurious diff), and secrets are
// never returned by the API so Extract must never touch them.
type StateCheck struct {
	Name  string
	State func() cc.State
	Check func(*testing.T, cc.State)
}

// MappingSuite drives the mapping beats every config-integration variant shares: BuildPayload
// populates the envelope + config, Extract maps the computed envelope fields (echoing any
// server-returned config), and a zero-value config never clobbers planned state. The envelope
// plumbing (template/enabled/default/business_units, diags) is asserted here once; per-variant
// suites only describe their config fields.
type MappingSuite[C any] struct {
	// BuildPayload / Extract are the variant's package-level mapping funcs — the same symbols
	// its production Spec is wired with.
	BuildPayload func(context.Context, cc.State, *diag.Diagnostics) api_client.ConfigEnvelope[C]
	Extract      func(*api_client.ConfigEnvelope[C], cc.State, *diag.Diagnostics) cc.APIObject

	// TemplateName is written into the built payload's common fields and asserted back.
	TemplateName string
	// SupportsBusinessUnits toggles the BU fixtures and assertions: forwarding on build,
	// null-set omission, and carry-through on extract. Non-BU variants are asserted to keep
	// business_units nil everywhere.
	SupportsBusinessUnits bool

	// FilledState returns a state with every variant attribute populated; the harness fills
	// the common fields itself.
	FilledState func() cc.State
	// CheckConfig asserts the built payload's config section matches FilledState.
	CheckConfig func(*testing.T, C)

	// EchoConfig is the config section of the fake API response in the extract beat.
	// CheckEchoed (optional) asserts the echoed values landed in EchoState.
	EchoConfig  C
	EchoState   func() cc.State
	CheckEchoed func(*testing.T, cc.State)

	// ZeroConfigChecks each run Extract with a zero-value config against a planned state and
	// assert the planned values survive.
	ZeroConfigChecks []StateCheck
}

// RunMappingSuite runs the shared beats as subtests.
func RunMappingSuite[C any](t *testing.T, s MappingSuite[C]) {
	t.Run("build payload populates all fields", s.checkBuild)
	if s.SupportsBusinessUnits {
		t.Run("null business_units omitted", s.checkNullBUsOmitted)
	}
	t.Run("extract maps computed fields", s.checkExtract)
	for _, chk := range s.ZeroConfigChecks {
		t.Run("zero-config extract: "+chk.Name, func(t *testing.T) {
			o := &api_client.ConfigEnvelope[C]{ID: "uuid", TemplateName: "t"}
			st := chk.State()
			var diags diag.Diagnostics
			s.Extract(o, st, &diags)
			requireNoDiags(t, diags)
			chk.Check(t, st)
		})
	}
}

// buildFrom fills the common fields on a fresh FilledState and runs BuildPayload.
func (s MappingSuite[C]) buildFrom(t *testing.T, enabled, deflt bool, bus types.Set) api_client.ConfigEnvelope[C] {
	t.Helper()
	st := s.FilledState()
	c := st.GetCommon()
	c.TemplateName = types.StringValue(s.TemplateName)
	c.IsEnabled = types.BoolValue(enabled)
	c.IsDefault = types.BoolValue(deflt)
	c.BusinessUnits = bus
	st.SetCommon(*c)

	var diags diag.Diagnostics
	p := s.BuildPayload(context.Background(), st, &diags)
	requireNoDiags(t, diags)
	return p
}

// flagRounds flips is_enabled / is_default across runs so a hardcoded flag value cannot pass.
var flagRounds = []struct{ enabled, deflt bool }{{true, false}, {false, true}}

func (s MappingSuite[C]) checkBuild(t *testing.T) {
	bus := types.SetNull(types.StringType)
	if s.SupportsBusinessUnits {
		bus = StringSet(t, suiteBUs...)
	}
	for _, round := range flagRounds {
		p := s.buildFrom(t, round.enabled, round.deflt, bus)
		if p.TemplateName != s.TemplateName {
			t.Errorf("template_name mismatch: %q", p.TemplateName)
		}
		if p.IsEnabled != round.enabled || p.IsDefault != round.deflt {
			t.Errorf("enabled/default mismatch: enabled=%v default=%v, want %v/%v",
				p.IsEnabled, p.IsDefault, round.enabled, round.deflt)
		}
		s.assertBUs(t, p.BusinessUnits)
		s.CheckConfig(t, p.Config)
	}
}

// A null business_units set must produce a nil slice so `omitempty` drops the field entirely
// (matches the UI: unset, not an empty array).
func (s MappingSuite[C]) checkNullBUsOmitted(t *testing.T) {
	p := s.buildFrom(t, true, false, types.SetNull(types.StringType))
	if p.BusinessUnits != nil {
		t.Errorf("null business_units must serialize as nil, got %v", p.BusinessUnits)
	}
}

func (s MappingSuite[C]) checkExtract(t *testing.T) {
	for _, round := range flagRounds {
		o := &api_client.ConfigEnvelope[C]{
			ID:           "uuid-1",
			TemplateName: s.TemplateName,
			IsEnabled:    round.enabled,
			IsDefault:    round.deflt,
			Config:       s.EchoConfig,
		}
		if s.SupportsBusinessUnits {
			o.BusinessUnits = suiteBUs
		}
		st := s.EchoState()
		var diags diag.Diagnostics
		got := s.Extract(o, st, &diags)
		requireNoDiags(t, diags)
		if got.ID != o.ID || got.TemplateName != o.TemplateName {
			t.Errorf("id/template mismatch: %+v", got)
		}
		if got.IsEnabled != round.enabled || got.IsDefault != round.deflt {
			t.Errorf("enabled/default mismatch: %+v, want %v/%v", got, round.enabled, round.deflt)
		}
		s.assertBUs(t, got.BusinessUnits)
		if s.CheckEchoed != nil {
			s.CheckEchoed(t, st)
		}
	}
}

// assertBUs checks a payload/object business_units slice against the suite fixture: BU-capable
// variants must carry it verbatim, non-BU variants must keep it nil.
func (s MappingSuite[C]) assertBUs(t *testing.T, got []string) {
	t.Helper()
	if s.SupportsBusinessUnits {
		if !SameElements(got, suiteBUs) {
			t.Errorf("business_units mismatch: %v", got)
		}
	} else if got != nil {
		t.Errorf("non-BU variant must not carry business_units, got %v", got)
	}
}

// AssertEq reports an error when got != want, prefixed with the field name. It keeps
// per-variant suite closures free of if/Errorf boilerplate.
func AssertEq(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s mismatch: got %q, want %q", name, got, want)
	}
}

func requireNoDiags(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
}
