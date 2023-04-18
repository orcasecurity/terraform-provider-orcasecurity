package api_client

type ComplianceFrameworkSection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ComplianceFramework struct {
	ID       string                       `json:"framework_id"`
	Name     string                       `json:"display_name"`
	Sections []ComplianceFrameworkSection `json:"sections"`
}

func (client *APIClient) GetCustomFrameworks() ([]ComplianceFramework, error) {
	type responseType struct {
		Data struct {
			Frameworks []ComplianceFramework `json:"frameworks"`
		} `json:"data"`
	}
	res, err := client.Get("/api/compliance/catalog?custom=true")
	if err != nil {
		return nil, err
	}
	response := responseType{}
	if err = res.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data.Frameworks, nil
}

func (client *APIClient) GetAlertCategories() ([]string, error) {
	type responseType struct {
		Data []string `json:"data"`
	}
	res, err := client.Get("/api/alerts/catalog/category")
	if err != nil {
		return nil, err
	}

	response := responseType{}
	if err = res.ReadJSON(&response); err != nil {
		return nil, err
	}
	return response.Data, nil
}
