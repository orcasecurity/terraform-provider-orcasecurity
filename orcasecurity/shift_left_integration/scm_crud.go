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

// AdoptWriteRequest carries the inputs for the shared adopt-existing
// Create/Update path (WriteAdopted / AdoptWrite).
type AdoptWriteRequest[T any] struct {
	// Get loads the live unit, Update PUTs the adopted body, Snapshot extracts
	// the adoptable fields from the live unit.
	Get      func() (*T, error)
	Update   func(api_client.ScmInstallationUpdate) (*T, error)
	Snapshot func(*T) ExistingUnit

	// Plan values; unset fields are hydrated from the live unit by Adopt.
	PlanMode     types.String
	PlanDefault  types.Bool
	PlanPolicies types.Set
	PlanConfig   *ConfigSettingsModel
	Project      ProjectIntent

	// Error copy used by AdoptWrite's diagnostics.
	Labels          AdoptLabels
	NotFoundMsg     string
	WriteErrorTitle string
}

// WriteAdopted is the shared Create/Update path for adopt-existing SCM resources:
// load the live unit, Adopt plan/config over it, PUT, return the refreshed unit.
func WriteAdopted[T any](req AdoptWriteRequest[T]) (*T, error) {
	current, err := req.Get()
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrUnitNotFound
	}
	ad := Adopt(req.PlanMode, req.PlanDefault, req.PlanPolicies, req.PlanConfig, req.Project, req.Snapshot(current))
	return req.Update(ad.Body)
}
