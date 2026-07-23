package api_client

import (
	"encoding/json"
	"fmt"
)

// scmEnvelope is the common enveloped list response for shift-left SCM endpoints.
type scmEnvelope[T any] struct {
	TotalItems int `json:"total_items"`
	Data       []T `json:"data"`
}

// scmInstallationID is the minimal shape needed to drive the
// installations -> integrated units fan-out.
type scmInstallationID struct {
	ID string `json:"id"`
}

// scmUnit is implemented by the SCM unit DTOs (GithubInstallation,
// GitlabGroup, AzureDevopsAccount, BitbucketAccount) so the shared
// list/find/update helpers can match a unit by id and stamp the parent
// installation id onto it (the per-installation unit lists return
// installation_id as null, so the client must fill it in).
type scmUnit interface {
	unitID() string
	stampInstallationID(string)
}

// invalidateScmListCache drops cached list pages so the next Get/List after a
// write re-fetches. Safe/no-op when unused.
func (client *APIClient) invalidateScmListCache() {
	client.scmListCache.Range(func(key, _ any) bool {
		client.scmListCache.Delete(key)
		return true
	})
}

// listScmUnitsByInstallation fans out across every installation: it lists the
// installations at installationsPath, then lists each installation's
// integrated units at unitsPath(installationID), stamping the installation id
// onto each unit. This is the only way to obtain the installation_id needed
// to drive a per-unit config resource for_each; the global list endpoints
// omit it.
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

// findScmUnit pages through unitsPath and returns the unit with the given id
// (stamped with installationID), or nil when absent so callers treat a
// missing unit as remote drift. Reads use list-filter because the API defines
// no single-unit GET routes for SCM units.
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

// updateScmUnit PUTs body to updatePath, invalidates the list cache, and
// returns the refreshed unit read back via findScmUnit.
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

// getAllScmPages fetches every page of an enveloped {total_items,data} list.
// basePath must already include a leading "/api" and no query string.
//
// Uses limit/start_at_index rather than limit/offset: the shift-left list
// paginator only honors `start_at_index` (the `offset` param is ignored —
// confirmed live on the projects endpoint), the same convention used by
// /api/automations (see ListAutomationsV2).
//
// Results are cached on the client for the lifetime of an apply/refresh until
// invalidateScmListCache is called (after every SCM PUT). That avoids O(n)
// full-list re-fetches when many SCM resources refresh the same list.
func getAllScmPages[T any](client *APIClient, basePath string) ([]T, error) {
	if cached, ok := client.scmListCache.Load(basePath); ok {
		if pages, ok := cached.([]T); ok {
			return pages, nil
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
			client.scmListCache.Store(basePath, all)
			return all, nil
		}
	}
}
