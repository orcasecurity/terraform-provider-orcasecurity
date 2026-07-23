package api_client

import (
	"encoding/json"
	"fmt"
)

type scmEnvelope[T any] struct {
	TotalItems int `json:"total_items"`
	Data       []T `json:"data"`
}

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
		for i := range units {
			PT(&units[i]).stampInstallationID(inst.ID)
		}
		all = append(all, units...)
	}
	return all, nil
}

// findScmUnit uses list-filter; the API defines no single-unit GET routes for SCM units.
func findScmUnit[T any, PT interface {
	*T
	scmUnit
}](client *APIClient, unitsPath, installationID, unitID string) (*T, error) {
	all, err := getAllScmPages[T](client, unitsPath)
	if err != nil {
		return nil, err
	}
	for i := range all {
		pt := PT(&all[i])
		if pt.unitID() == unitID {
			pt.stampInstallationID(installationID)
			return &all[i], nil
		}
	}
	return nil, nil
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
	if cached, ok := client.scmListCache.Load(basePath); ok {
		if entry, ok := cached.(scmCacheEntry); ok && entry.gen == startGen {
			if pages, ok := entry.data.([]T); ok {
				return pages, nil
			}
		}
	}

	const pageLimit = 200
	var all []T
	for {
		resp, err := client.Get(fmt.Sprintf("%s?limit=%d&start_at_index=%d", basePath, pageLimit, len(all)))
		if err != nil {
			return nil, err
		}
		var env scmEnvelope[T]
		if err := json.Unmarshal(resp.Body(), &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if len(env.Data) == 0 || len(all) >= env.TotalItems {
			// Only cache if no invalidation happened during the fetch; otherwise
			// these pages predate the write and must not be resurrected.
			if client.scmListGen.Load() == startGen {
				client.scmListCache.Store(basePath, scmCacheEntry{gen: startGen, data: all})
			}
			return all, nil
		}
	}
}
