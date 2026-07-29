package api_client

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
	return findScmUnit[T, PT](client, unitsPath, installationID, unitID)
}

// getAllScmPages uses limit/start_at_index (offset is ignored on shift-left lists).
// filters narrow the result set server-side; pass nil to fetch the whole list.
//
// Reads are not cached. The only lists that grow with tenant size are the
// integrated repositories, and every Find*Repository now narrows those
// server-side: GitLab and Azure DevOps to a single row, GitHub to one repository
// name, Bitbucket to one account. The unit lists are already scoped to a single
// installation. A cache would have to be invalidated after each of the dozen SCM
// write paths, and the reads it could still serve are the ones a write is about
// to invalidate anyway.
func getAllScmPages[T any](client *APIClient, basePath string, filters listFilters) ([]T, error) {
	const pageLimit = 200
	const maxScmPages = 500 // backstop against an inflated/bogus total_items with full pages
	return paginateOffset[T](client, basePath, filters, pageLimit, maxScmPages)
}
