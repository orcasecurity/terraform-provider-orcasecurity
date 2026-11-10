package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Vulnerability struct {
	CVEID          string   `json:"cve_id"`
	Description    string   `json:"description"`
	Expiration     string   `json:"expiration,omitempty"`
	Disabled       bool     `json:"disabled"`
	RepositoryURLs []string `json:"repositories_urls,omitempty"`
}

type Project struct {
	ProjectID   string `json:"id"`
	ProjectName string `json:"name"`
	ProjectKey  string `json:"key"`
}

type ShiftLeftCveExceptionList struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// For some orgs the API never stores list-level disabled=true (it disables every entry instead)
	// and omits the field from responses; Disabled is then nil.
	Disabled        *bool           `json:"disabled"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
	Projects        []Project       `json:"projects,omitempty"`
}

func (client *APIClient) GetShiftLeftCveExceptionList(id string) (*ShiftLeftCveExceptionList, error) {
	resp, err := client.Get(fmt.Sprintf("/api/shiftleft/exceptions/%s/", id))
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := ShiftLeftCveExceptionList{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DoesShiftLeftCveExceptionListExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/shiftleft/exceptions/%s/", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) createShiftLeftCveExceptionList(data ShiftLeftCveExceptionList) (*ShiftLeftCveExceptionList, error) {
	resp, err := client.PostCreate("/api/shiftleft/exceptions/", data)
	if err != nil {
		return nil, err
	}

	if !resp.IsOk() {
		return nil, nil
	}

	response := ShiftLeftCveExceptionList{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) UpdateShiftLeftCveExceptionList(ID string, data ShiftLeftCveExceptionList) (*ShiftLeftCveExceptionList, error) {
	resp, err := client.Put(fmt.Sprintf("/api/shiftleft/exceptions/%s/", ID), data)
	if err != nil {
		return nil, err
	}

	response := ShiftLeftCveExceptionList{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (client *APIClient) DeleteShiftLeftCveExceptionList(ID string) error {
	_, err := client.Delete(fmt.Sprintf("/api/shiftleft/exceptions/%s/", ID))
	return err
}

type shiftLeftExceptionListSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (client *APIClient) ListShiftLeftCveExceptionListIDsByName(name string) (map[string]bool, error) {
	filters := listFilters{"search": strings.TrimSpace(name), "search_fields": "exception_name"}
	all, err := getAllScmPages[shiftLeftExceptionListSummary](client, "/api/shiftleft/exceptions/", filters)
	if err != nil {
		return nil, err
	}
	return idsWhere(all, func(e *shiftLeftExceptionListSummary) string { return e.ID },
		func(e *shiftLeftExceptionListSummary) bool { return sameName(e.Name, name) }), nil
}

func (client *APIClient) ShiftLeftCveExceptionListCreateOp(data ShiftLeftCveExceptionList) CreateOp[ShiftLeftCveExceptionList] {
	return CreateOp[ShiftLeftCveExceptionList]{
		What:   fmt.Sprintf("shift-left exception list named %q", data.Name),
		ID:     func(o *ShiftLeftCveExceptionList) string { return o.ID },
		Create: func() (*ShiftLeftCveExceptionList, error) { return client.createShiftLeftCveExceptionList(data) },
		IDs:    func() (map[string]bool, error) { return client.ListShiftLeftCveExceptionListIDsByName(data.Name) },
		Get:    client.GetShiftLeftCveExceptionList,
	}
}
