package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ErrUnitNotFound = errors.New("scm unit not found")

type Commoner interface {
	Common() api_client.ScmUnitCommonFields
}

func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return tfconv.StringSliceToSet(api_client.PolicyRefIDs(refs))
}
