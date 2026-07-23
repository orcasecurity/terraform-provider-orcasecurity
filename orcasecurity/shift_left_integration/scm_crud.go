package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ErrUnitNotFound is returned by WriteAdopted when get() yields a nil unit.
var ErrUnitNotFound = errors.New("scm unit not found")

// ExistingFromAPI builds an ExistingUnit from the common SCM API fields.
func ExistingFromAPI(
	mode string,
	defaultPolicies bool,
	policies []api_client.ScmPolicyRef,
	project *api_client.ScmProjectRef,
	cfg api_client.ShiftLeftConfigSettings,
) ExistingUnit {
	return ExistingUnit{
		InstallationMode: mode,
		DefaultPolicies:  defaultPolicies,
		PolicyIDs:        api_client.PolicyRefIDs(policies),
		ConfigSettings:   cfg,
		ProjectID:        api_client.ProjectRefID(project),
	}
}

// PolicyIDsFromRefs converts API policy references into a policies_ids set.
func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return PolicyIDsToSet(api_client.PolicyRefIDs(refs))
}

// WriteAdopted is the shared Create/Update path for adopt-existing SCM resources:
// load the live unit, Adopt plan/config over it, PUT, return the refreshed unit.
func WriteAdopted[T any](
	get func() (*T, error),
	update func(api_client.ScmInstallationUpdate) (*T, error),
	snapshot func(*T) ExistingUnit,
	planMode types.String,
	planDefault types.Bool,
	planPolicies types.Set,
	planConfig *ConfigSettingsModel,
	project ProjectIntent,
) (*T, error) {
	current, err := get()
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrUnitNotFound
	}
	ad := Adopt(planMode, planDefault, planPolicies, planConfig, project, snapshot(current))
	return update(ad.Body)
}
