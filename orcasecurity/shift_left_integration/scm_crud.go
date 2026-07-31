package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ErrUnitNotFound = errors.New("scm unit not found")

// ScmUnit is satisfied by every concrete SCM unit API type via the embedded
// api_client.ScmUnitCommonFields.
type ScmUnit interface {
	Common() api_client.ScmUnitCommonFields
}

func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return PolicyIDsToSet(api_client.PolicyRefIDs(refs))
}
