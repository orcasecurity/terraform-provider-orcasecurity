package api_client

import "fmt"

// Shared CRUD plumbing for SCM parent installations (GitLab, Bitbucket,
// Azure DevOps). The API defines no single-item GET route for installations,
// so reads go through the cached list and filter by id.

// installationIDer is implemented by the parent-installation DTOs so the
// shared helpers can match a row by id.
type installationIDer interface {
	installationID() string
}

// findScmInstallation reads via list-filter. Returns nil when absent.
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

// createScmInstallation POSTs and decodes the created row (all providers echo
// the full serializer, including id).
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

// patchScmInstallationAndReread PATCHes and reads the row back via the list
// (for providers whose PATCH returns an empty body: GitLab, Azure DevOps).
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

// patchScmInstallation PATCHes and decodes the response body (for providers
// whose PATCH echoes the full serializer: Bitbucket).
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

// deleteScmPathIgnoring404 DELETEs path and treats 404 as success so Terraform
// destroy stays idempotent when the unit is already gone.
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
