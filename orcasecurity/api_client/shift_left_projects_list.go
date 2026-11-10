package api_client

import (
	"fmt"
	"strings"
)

type ShiftLeftProjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

func (client *APIClient) ListShiftLeftProjects() ([]ShiftLeftProjectSummary, error) {
	return getAllScmPages[ShiftLeftProjectSummary](client, "/api/shiftleft/projects/", nil)
}

// Projects are unique per org by name and by key; key is the identity, name is the fallback when unset.
// Lists all instead of ?search=<key>&search_fields=project_key: live, that exact search kept
// answering empty for over 20s after the project was created, so recovery never saw it.
func (client *APIClient) ListShiftLeftProjectIDsByKey(key, name string) (map[string]bool, error) {
	match := func(p *ShiftLeftProjectSummary) bool { return sameName(p.Key, key) }
	if strings.TrimSpace(key) == "" {
		match = func(p *ShiftLeftProjectSummary) bool { return sameName(p.Name, name) }
	}
	all, err := client.ListShiftLeftProjects()
	if err != nil {
		return nil, err
	}
	return idsWhere(all, func(p *ShiftLeftProjectSummary) string { return p.ID }, match), nil
}

func (client *APIClient) ShiftLeftProjectCreateOp(data ShiftLeftProject) CreateOp[ShiftLeftProject] {
	what := fmt.Sprintf("shift-left project with key %q", data.Key)
	if strings.TrimSpace(data.Key) == "" {
		what = fmt.Sprintf("shift-left project named %q", data.Name)
	}
	return CreateOp[ShiftLeftProject]{
		What:   what,
		Create: func() (*ShiftLeftProject, error) { return client.CreateShiftLeftProject(data) },
		IDs:    func() (map[string]bool, error) { return client.ListShiftLeftProjectIDsByKey(data.Key, data.Name) },
		Get:    client.GetShiftLeftProject,
	}
}
