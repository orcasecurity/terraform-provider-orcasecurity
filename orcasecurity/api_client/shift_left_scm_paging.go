package api_client

type scmInstallationID struct {
	ID string `json:"id"`
}

type scmUnit interface {
	unitID() string
	stampInstallationID(string)
}

func (client *APIClient) InvalidateScmListCache() {
	client.invalidateScmListCache()
}

func (client *APIClient) invalidateScmListCache() {
	// Bump the generation first so any fetch already in flight stores under the
	// old generation and fails the generation check on the next read.
	client.scmListGen.Add(1)
	client.scmListCache.Range(func(key, _ any) bool {
		client.scmListCache.Delete(key)
		return true
	})
}

// scmCacheEntry tags cached pages with the generation seen when their fetch began.
type scmCacheEntry struct {
	gen  uint64
	data any
}

func loadScmListCache[T any](client *APIClient, basePath string, gen uint64) ([]T, bool) {
	cached, ok := client.scmListCache.Load(basePath)
	if !ok {
		return nil, false
	}
	entry, ok := cached.(scmCacheEntry)
	if !ok || entry.gen != gen {
		return nil, false
	}
	pages, ok := entry.data.([]T)
	if !ok {
		return nil, false
	}
	// Callers copy before stamping; no clone here.
	return pages, true
}

func storeScmListCacheIfCurrent[T any](client *APIClient, basePath string, startGen uint64, all []T) {
	if client.scmListGen.Load() == startGen {
		client.scmListCache.Store(basePath, scmCacheEntry{gen: startGen, data: all})
	}
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
	installations, err := getAllScmPages[scmInstallationID](client, installationsPath)
	if err != nil {
		return nil, err
	}
	var all []T
	for _, inst := range installations {
		units, err := getAllScmPages[T](client, unitsPath(inst.ID))
		if err != nil {
			return nil, err
		}
		// Stamp into all[], not the cached units slice (may be shared).
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
	all, err := getAllScmPages[T](client, unitsPath)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if match(&all[i]) {
			unit := all[i] // copy before stamp; cached slice is shared
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
	client.invalidateScmListCache()
	return findScmUnit[T, PT](client, unitsPath, installationID, unitID)
}

// getAllScmPages uses limit/start_at_index (offset is ignored on shift-left lists).
// Results are cached until invalidateScmListCache runs after SCM writes.
func getAllScmPages[T any](client *APIClient, basePath string) ([]T, error) {
	// Snapshot the generation before fetching. A concurrent write invalidates by
	// bumping this generation; a store guarded by the snapshot is dropped when it
	// no longer matches, so a stale read cannot repopulate the cache.
	startGen := client.scmListGen.Load()
	if pages, ok := loadScmListCache[T](client, basePath, startGen); ok {
		return pages, nil
	}

	// Collapse concurrent fetches of the same path into one request.
	v, err, _ := client.scmListFlight.Do(basePath, func() (any, error) {
		return fetchAllScmPages[T](client, basePath, startGen)
	})
	if err != nil {
		return nil, err
	}
	if pages, ok := v.([]T); ok {
		return pages, nil
	}
	// A concurrent fetch for a different T raced on this basePath key; fetch directly.
	return fetchAllScmPages[T](client, basePath, startGen)
}

func fetchAllScmPages[T any](client *APIClient, basePath string, startGen uint64) ([]T, error) {
	const pageLimit = 200
	const maxScmPages = 500 // backstop against an inflated/bogus total_items with full pages
	all, err := paginateOffset[T](client, basePath, pageLimit, maxScmPages)
	if err != nil {
		return nil, err
	}
	// Only cache if no invalidation happened during the fetch; otherwise these
	// pages predate the write and must not be resurrected.
	storeScmListCacheIfCurrent(client, basePath, startGen, all)
	return all, nil
}
