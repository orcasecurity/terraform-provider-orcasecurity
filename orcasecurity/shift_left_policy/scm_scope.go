package shift_left_policy

import (
	"fmt"
	"sort"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

// verifyScmPostureScopeApplied checks that every scope id requested in plan is present
// on the API policy. The backend accepts unknown/unlinked UUIDs with HTTP 201 and
// returns empty scope lists, which would otherwise settle in state (via
// stateFromPlanAfterWrite) and then perpetual-drift on the next refresh.
func verifyScmPostureScopeApplied(plan *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy) error {
	if plan == nil || apiPolicy == nil {
		return nil
	}
	if plan.Type.ValueString() != "scm_posture" {
		return nil
	}
	if plan.ScmPosture == nil || len(plan.ScmPosture.Scope) == 0 {
		return nil
	}

	want := scopeIDSets(plan.ScmPosture.Scope)
	got := scopeIDSets(buildScmPostureBlock(apiPolicy, nil).Scope)

	var missing []string
	for key, wantIDs := range want {
		gotIDs := got[key]
		for id := range wantIDs {
			if !gotIDs[id] {
				missing = append(missing, fmt.Sprintf("%s=%s", key, id))
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf(
		"requested scm_posture scope was not applied by the API (%s); "+
			"the installation/unit ids may be invalid, unlinked, or already assigned to another custom policy",
		strings.Join(missing, ", "),
	)
}

func scopeIDSets(entries []scmScopeEntryModel) map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(entries))
	for _, entry := range entries {
		key := entry.Key.ValueString()
		if key == "" {
			continue
		}
		ids := out[key]
		if ids == nil {
			ids = map[string]bool{}
			out[key] = ids
		}
		for _, id := range stringSliceFromTypes(entry.Ids) {
			if id != "" {
				ids[id] = true
			}
		}
	}
	return out
}
