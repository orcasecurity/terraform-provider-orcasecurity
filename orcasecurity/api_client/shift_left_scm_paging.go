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

// scmUnitNameFilter narrows a unit list to rows whose name matches the term. Every SCM's unit list
// exposes exactly one searchable name field (the GitLab group name, the Bitbucket account slug, the
// Azure organization name), reached through the shared `name` alias. It is a hint only: the search is
// a partial match, so the caller still has to match exactly.
func scmUnitNameFilter(name string) listFilters {
	if name == "" {
		return nil
	}
	return listFilters{"search": name, "search_fields": "name"}
}

// findScmUnitBy list-filters; the API has no single-unit GET route for SCM units, and the unit lists
// accept no filter on the unit id, so a lookup by id has to walk every page. Callers looking a unit
// up by name pass scmUnitNameFilter to narrow that walk. A filtered miss falls back to the full scan
// because the filter searches one specific name field: reporting absence on a filter mismatch would
// make Terraform treat a live unit as deleted.
func findScmUnitBy[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, unitsPath, installationID string, filters listFilters, match func(*T) bool) (*T, error) {
	unit, err := findScmUnitOnPages[T, PT](client, unitsPath, installationID, filters, match)
	if err != nil || unit != nil || filters == nil {
		return unit, err
	}
	return findScmUnitOnPages[T, PT](client, unitsPath, installationID, nil, match)
}

func findScmUnitOnPages[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, unitsPath, installationID string, filters listFilters, match func(*T) bool) (*T, error) {
	all, err := getAllScmPages[T](client, unitsPath, filters)
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
	return findScmUnitBy[T, PT](client, unitsPath, installationID, nil, func(u *T) bool {
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
