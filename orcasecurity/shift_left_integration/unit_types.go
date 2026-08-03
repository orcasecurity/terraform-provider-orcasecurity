package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ErrUnitNotFound is returned by unit lookups (and AdoptedUnitOps.Delete
// implementations) when the unit no longer exists remotely.
var ErrUnitNotFound = errors.New("scm unit not found")

// Commoner is satisfied by every concrete SCM unit API type via the embedded
// api_client.ScmUnitCommonFields.
type Commoner interface {
	Common() api_client.ScmUnitCommonFields
}

func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return tfconv.StringSliceToSet(api_client.PolicyRefIDs(refs))
}
