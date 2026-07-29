package api_client

import (
	"encoding/json"
	"fmt"
)

// No single-item GET route for installations; reads scan the list.

type installationIDer interface {
	installationID() string
}

func findScmInstallation[T any, PT interface {
	*T
	installationIDer
}](client *APIClient, listPath, id string) (*T, error) {
	all, err := getAllScmPages[T](client, listPath, nil)
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
	if len(resp.Body()) == 0 {
		return nil, fmt.Errorf("create at %s returned an empty body; installation may exist in Orca but is untracked, run terraform refresh", listPath)
	}
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
	_, err := client.Patch(fmt.Sprintf("%s%s/", listPath, id), body)
	if err != nil {
		return nil, err
	}
	return findScmInstallation[T, PT](client, listPath, id)
}

func patchScmInstallation[T any](client *APIClient, listPath, id string, body any) (*T, error) {
	resp, err := client.Patch(fmt.Sprintf("%s%s/", listPath, id), body)
	if err != nil {
		return nil, err
	}
	if len(resp.Body()) == 0 {
		return nil, fmt.Errorf("update at %s%s/ returned an empty body; run terraform refresh to reconcile", listPath, id)
	}
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
	if resp != nil && (resp.StatusCode() == 404 || scmDeleteAlreadyInert(resp)) {
		return nil
	}
	return err
}

// A suspended GitHub App installation returns 400 github_installation_suspended
// instead of 404/204; the integration is already inert, so treat it as deleted.
func scmDeleteAlreadyInert(resp *APIResponse) bool {
	if resp.StatusCode() != 400 {
		return false
	}
	var body struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(resp.Body(), &body); err != nil {
		return false
	}
	return body.ErrorCode == "github_installation_suspended"
}
