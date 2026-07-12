package api_client

import (
	"fmt"
)

// Jira Cloud credentials are created through the OAuth flow in the Orca UI, not via the API,
// so there is no create/update/delete here — only a read of the connected Jira Cloud sites.
// Orca exposes them at GET /api/jira/resources, distinct from the generic
// /api/external_service/resources endpoint used by token-based integrations.
const jiraCloudResourcesPath = "/api/jira/resources"

// JiraCloudResource is one connected Jira Cloud site. ID is the Jira cloud id used as
// `resource_id` on orcasecurity_integration_jira_cloud_template.
type JiraCloudResource struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	URL  string `json:"url"`
}

func (client *APIClient) ListJiraCloudResources() ([]JiraCloudResource, error) {
	resp, err := client.Get(jiraCloudResourcesPath)
	if err != nil {
		return nil, err
	}
	type wrapped struct {
		Status string `json:"status"`
		Data   struct {
			Resources []JiraCloudResource `json:"resources"`
		} `json:"data"`
	}
	response := wrapped{}
	if err := resp.ReadJSON(&response); err != nil {
		return nil, fmt.Errorf("failed to decode Jira Cloud resources response: %w", err)
	}
	return response.Data.Resources, nil
}

func (client *APIClient) GetJiraCloudResourceByName(name string) (*JiraCloudResource, error) {
	all, err := client.ListJiraCloudResources()
	if err != nil {
		return nil, err
	}
	var matches []JiraCloudResource
	for _, item := range all {
		if item.Name == name {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple Jira Cloud resources named %q — provide the ID instead", name)
	}
	return &matches[0], nil
}
