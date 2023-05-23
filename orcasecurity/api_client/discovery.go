package api_client

type SonarQueryResultRow struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func (client *APIClient) ExecuteSonarQuery(query map[string]interface{}, limit int64, startIndex int64) ([]SonarQueryResultRow, error) {
	type payload struct {
		Query        map[string]interface{} `json:"query"`
		Limit        int64                  `json:"limit"`
		StartAtIndex int64                  `json:"start_at_index"`
	}
	resp, err := client.Post("/api/sonar/query", payload{
		Query: query, Limit: limit, StartAtIndex: startIndex,
	})
	if err != nil {
		return nil, err
	}

	type responseType struct {
		Data []SonarQueryResultRow `json:"data"`
	}

	response := responseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}
