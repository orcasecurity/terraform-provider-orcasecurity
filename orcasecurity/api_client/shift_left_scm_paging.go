package api_client

import "fmt"

type scmInstallationID struct {
	ID string `json:"id"`
}

type scmUnit interface {
	unitID() string
	stampInstallationID(string)
}

// listScmUnitsByInstallation is required to obtain installation_id for for_each; global lists omit it.
func listScmUnitsByInstallation[T any, PT interface {
	*T
	scmUnit
}](
	client *APIClient,
	installationsPath string,
	unitsPath func(installationID string) string,
) ([]T, error) {
	installations, err := getAllScmPages[scmInstallationID](client, installationsPath, nil)
	if err != nil {
		return nil, err
	}
	var all []T
	for _, inst := range installations {
		units, err := getAllScmPages[T](client, unitsPath(inst.ID), nil)
		if err != nil {
			return nil, err
		}
		start := len(all)
		all = append(all, units...)
		for i := start; i < len(all); i++ {
			PT(&all[i]).stampInstallationID(inst.ID)
		}
	}
	return all, nil
}

// findScmUnitBy list-filters; the API has no single-unit GET route for SCM units.
func findScmUnitBy[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, unitsPath, installationID string, match func(*T) bool) (*T, error) {
	all, err := getAllScmPages[T](client, unitsPath, nil)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if match(&all[i]) {
			unit := all[i]
			PT(&unit).stampInstallationID(installationID)
			return &unit, nil
		}
	}
	return nil, nil
}

func findScmUnit[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, unitsPath, installationID, unitID string) (*T, error) {
	return findScmUnitBy[T, PT](client, unitsPath, installationID, func(u *T) bool {
		return PT(u).unitID() == unitID
	})
}

func updateScmUnit[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, updatePath, unitsPath, installationID, unitID string, body ScmInstallationUpdate) (*T, error) {
	if _, err := client.Put(updatePath, body); err != nil {
		return nil, err
	}
	unit, err := findScmUnit[T, PT](client, unitsPath, installationID, unitID)
	if err != nil {
		return nil, err
	}
	// The write succeeded, so a missed read-back is not "the unit is gone". Returning
	// (nil, nil) here would be read as absence and drop state for a unit we just updated.
	if unit == nil {
		return nil, fmt.Errorf("%s: update succeeded but the unit could not be read back; run terraform refresh", updatePath)
	}
	return unit, nil
}

// Shift-left lists paginate with start_at_index (offset ignored). Pass nil filters for full list.
func getAllScmPages[T any](client *APIClient, basePath string, filters listFilters) ([]T, error) {
	const pageLimit = 200
	const maxScmPages = 500 // backstop against an inflated/bogus total_items with full pages
	return paginateOffset[T](client, basePath, filters, pageLimit, maxScmPages)
}
