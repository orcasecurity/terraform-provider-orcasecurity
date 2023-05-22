package api_client

type Organization struct {
	ID   string `json:"organization_id"`
	Name string `json:"organization_name"`
}

func (client *APIClient) GetCurrentOrganization() (*Organization, error) {
	resp, err := client.Get("/api/user/action")
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data Organization `json:"data"`
	}

	response := responseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	return &response.Data, nil
}
