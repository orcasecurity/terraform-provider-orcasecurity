package api_client

import "fmt"

// No single-item GET route for installations; reads use the cached list.

type installationIDer interface {
	installationID() string
}

func findScmInstallation[T any, PT interface {
	*T
	installationIDer
}](client *APIClient, listPath, id string) (*T, error) {
	all, err := getAllScmPages[T](client, listPath)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if PT(&all[i]).installationID() == id {
			return &all[i], nil
		}
	}
	return nil, nil
}

func createScmInstallation[T any](client *APIClient, listPath string, body any) (*T, error) {
	resp, err := client.Post(listPath, body)
	if err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	created := new(T)
	if err := resp.ReadJSON(created); err != nil {
		return nil, err
	}
	return created, nil
}

// GitLab and Azure DevOps PATCH return an empty body; Bitbucket echoes the full serializer.
func patchScmInstallationAndReread[T any, PT interface {
	*T
	installationIDer
}](client *APIClient, listPath, id string, body any) (*T, error) {
	if _, err := client.Patch(fmt.Sprintf("%s%s/", listPath, id), body); err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	return findScmInstallation[T, PT](client, listPath, id)
}

func patchScmInstallation[T any](client *APIClient, listPath, id string, body any) (*T, error) {
	resp, err := client.Patch(fmt.Sprintf("%s%s/", listPath, id), body)
	if err != nil {
		return nil, err
	}
	client.invalidateScmListCache()
	updated := new(T)
	if err := resp.ReadJSON(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func deleteScmInstallation(client *APIClient, listPath, id string) error {
	return deleteScmPathIgnoring404(client, fmt.Sprintf("%s%s/", listPath, id))
}

// Treat 404 as success so destroy stays idempotent when the unit is already gone.
func deleteScmPathIgnoring404(client *APIClient, path string) error {
	resp, err := client.Delete(path)
	if resp != nil && resp.StatusCode() == 404 {
		client.invalidateScmListCache()
		return nil
	}
	if err == nil {
		client.invalidateScmListCache()
	}
	return err
}
