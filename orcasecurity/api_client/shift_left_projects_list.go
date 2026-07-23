package api_client

type ShiftLeftProjectSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// Uses start_at_index paging via shared SCM list helper.
func (client *APIClient) ListShiftLeftProjects() ([]ShiftLeftProjectSummary, error) {
	return getAllScmPages[ShiftLeftProjectSummary](client, "/api/shiftleft/projects/")
}
