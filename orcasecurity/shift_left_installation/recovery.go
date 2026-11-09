package shift_left_installation

import "terraform-provider-orcasecurity/orcasecurity/internal/createrecovery"

// installationIDsNamed adapts an installation list to the id set createrecovery diffs.
func installationIDsNamed[T any](list func() ([]T, error), idName func(*T) (string, string)) func(string) (map[string]bool, error) {
	return func(name string) (map[string]bool, error) {
		all, err := list()
		if err != nil {
			return nil, err
		}
		ids := map[string]bool{}
		for i := range all {
			if id, n := idName(&all[i]); createrecovery.SameName(n, name) {
				ids[id] = true
			}
		}
		return ids, nil
	}
}
