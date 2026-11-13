package integrations_common

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ValidateShiftleftGlobalScope mirrors the API rule for the RBAC access and invite
// payloads: shiftleft_global is a scope of its own and cannot be combined with
// all_cloud_accounts or with any of the list scopes. Unknown values are skipped,
// the API rejects the request if they resolve to a conflict at apply time.
func ValidateShiftleftGlobalScope(shiftleftGlobal types.Bool, allCloudAccounts types.Bool, lists map[string]types.List) diag.Diagnostics {
	if shiftleftGlobal.IsUnknown() || !shiftleftGlobal.ValueBool() {
		return nil
	}
	var conflicting []string
	if !allCloudAccounts.IsUnknown() && allCloudAccounts.ValueBool() {
		conflicting = append(conflicting, "all_cloud_accounts")
	}
	for name, list := range lists {
		if list.IsUnknown() || list.IsNull() || len(list.Elements()) == 0 {
			continue
		}
		conflicting = append(conflicting, name)
	}
	if len(conflicting) == 0 {
		return nil
	}
	sort.Strings(conflicting)
	var diags diag.Diagnostics
	diags.AddAttributeError(
		path.Root("shiftleft_global"),
		"Conflicting scope",
		fmt.Sprintf("shiftleft_global cannot be combined with %s: it grants the role on every Shift Left project and is a scope of its own.", strings.Join(conflicting, ", ")),
	)
	return diags
}
