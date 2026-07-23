package shift_left_integration

import (
	"errors"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ErrUnitNotFound = errors.New("scm unit not found")

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

func ExistingFromCommon(c api_client.ScmUnitCommonFields) ExistingUnit {
	return ExistingFromAPI(c.InstallationMode, c.DefaultPolicies, c.Policies, c.Project, c.ConfigSettings)
}

func PolicyIDsFromRefs(refs []api_client.ScmPolicyRef) types.Set {
	return PolicyIDsToSet(api_client.PolicyRefIDs(refs))
}

type AdoptWriteRequest[T any] struct {
	Get      func() (*T, error)
	Update   func(current *T, body api_client.ScmInstallationUpdate) (*T, error)
	Snapshot func(*T) ExistingUnit

	PlanMode     types.String
	PlanDefault  types.Bool
	PlanPolicies types.Set
	PlanConfig   *ConfigSettingsModel
	Project      ProjectIntent

	Labels          AdoptLabels
	NotFoundMsg     string
	WriteErrorTitle string
}

func WriteAdopted[T any](req AdoptWriteRequest[T]) (*T, error) {
	current, err := req.Get()
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrUnitNotFound
	}
	ad := Adopt(req.PlanMode, req.PlanDefault, req.PlanPolicies, req.PlanConfig, req.Project, req.Snapshot(current))
	return req.Update(current, ad.Body)
}
