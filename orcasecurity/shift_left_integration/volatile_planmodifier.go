package shift_left_integration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// volatile* plan modifiers are for Computed attributes the server may change as
// a side effect of other updates (repo enrollment, health flaps, scan-all
// progress).
//
// Behaviour:
//   - Compare Config.Raw to State.Raw (not Plan.Raw). Computed attributes are
//     null in config, so they do not block carry-forward the way an unknown
//     planned value would.
//   - If any configured value is unknown, we cannot prove "no write" — leave
//     the volatile attribute unknown. Treating unknown as "match" caused
//     known→unknown on apply re-plan when a cross-resource reference resolved
//     to a different writable value (Terraform forbids that transition).
//   - Null config leaves are omitted / computed-only — skipped.
//   - Known configured values must equal state; otherwise leave unknown so a
//     side-effecting write cannot produce an inconsistent apply result.
//   - When config matches, carry prior state forward so empty plans settle on
//     Terraform 1.0–1.3.

type volatileStringModifier struct{}

func VolatileString() planmodifier.String { return volatileStringModifier{} }

func (volatileStringModifier) Description(context.Context) string {
	return "Volatile: carried forward only when the plan has no writable changes; otherwise re-read after apply."
}
func (m volatileStringModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (volatileStringModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if carryVolatileForward(req.State.Raw, req.Config.Raw, req.PlanValue.IsUnknown()) {
		resp.PlanValue = req.StateValue
	}
}

type volatileInt64Modifier struct{}

func VolatileInt64() planmodifier.Int64 { return volatileInt64Modifier{} }

func (volatileInt64Modifier) Description(context.Context) string {
	return "Volatile: carried forward only when the plan has no writable changes; otherwise re-read after apply."
}
func (m volatileInt64Modifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (volatileInt64Modifier) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if carryVolatileForward(req.State.Raw, req.Config.Raw, req.PlanValue.IsUnknown()) {
		resp.PlanValue = req.StateValue
	}
}

// carryVolatileForward is true when prior state exists, the planned value is
// still unknown, and every known configured value matches state (no write).
func carryVolatileForward(state, config tftypes.Value, planValueUnknown bool) bool {
	if state.IsNull() || !planValueUnknown {
		return false
	}
	return writableConfigMatchesState(config, state)
}

// writableConfigMatchesState reports whether config declares no pending write
// relative to state. Unknown config → false. Null config leaves are skipped
// (omitted or Computed-only). Known config values must equal state.
func writableConfigMatchesState(config, state tftypes.Value) bool {
	if !config.IsKnown() {
		return false
	}
	if config.IsNull() {
		return true
	}
	if !state.IsKnown() || state.IsNull() {
		return false
	}

	switch {
	case config.Type().Is(tftypes.Object{}):
		var configObj, stateObj map[string]tftypes.Value
		if err := config.As(&configObj); err != nil {
			return false
		}
		if err := state.As(&stateObj); err != nil {
			return false
		}
		for k, cv := range configObj {
			sv, ok := stateObj[k]
			if !ok {
				if !cv.IsKnown() {
					return false
				}
				if cv.IsNull() {
					continue
				}
				return false
			}
			if !writableConfigMatchesState(cv, sv) {
				return false
			}
		}
		return true
	case config.Type().Is(tftypes.Map{}):
		var configMap, stateMap map[string]tftypes.Value
		if err := config.As(&configMap); err != nil {
			return false
		}
		if err := state.As(&stateMap); err != nil {
			return false
		}
		if len(configMap) != len(stateMap) {
			return false
		}
		for k, cv := range configMap {
			sv, ok := stateMap[k]
			if !ok {
				return false
			}
			if !writableConfigMatchesState(cv, sv) {
				return false
			}
		}
		return true
	case config.Type().Is(tftypes.List{}), config.Type().Is(tftypes.Tuple{}):
		var configList, stateList []tftypes.Value
		if err := config.As(&configList); err != nil {
			return false
		}
		if err := state.As(&stateList); err != nil {
			return false
		}
		if len(configList) != len(stateList) {
			return false
		}
		for i := range configList {
			if !writableConfigMatchesState(configList[i], stateList[i]) {
				return false
			}
		}
		return true
	case config.Type().Is(tftypes.Set{}):
		var configSet, stateSet []tftypes.Value
		if err := config.As(&configSet); err != nil {
			return false
		}
		if err := state.As(&stateSet); err != nil {
			return false
		}
		if len(configSet) != len(stateSet) {
			return false
		}
		used := make([]bool, len(stateSet))
		for _, cv := range configSet {
			if !cv.IsKnown() {
				return false
			}
			if cv.IsNull() {
				continue
			}
			matched := false
			for j, sv := range stateSet {
				if used[j] {
					continue
				}
				if writableConfigMatchesState(cv, sv) {
					used[j] = true
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	default:
		return config.Equal(state)
	}
}
