package api_client

// ShiftLeftProjectSummary is the minimal shape needed to enumerate every
// shift-left project for fleet-wide operations (e.g. bulk policy attach).
// The list endpoint returns a much richer object (policies, exceptions,
// config settings, ...); only id/name/key are surfaced here.
type ShiftLeftProjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ListShiftLeftProjects returns every shift-left project in the organization.
// The endpoint shares the standard shift-left list contract (limit +
// start_at_index paging, {total_items,data} envelope), so it pages through
// the shared SCM list helper and benefits from its per-apply cache.
func (client *APIClient) ListShiftLeftProjects() ([]ShiftLeftProjectSummary, error) {
	return getAllScmPages[ShiftLeftProjectSummary](client, "/api/shiftleft/projects/")
}
