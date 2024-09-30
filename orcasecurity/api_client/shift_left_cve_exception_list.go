package api_client

import (
	"encoding/json"
	"fmt"
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
	ID              string          `json:"id,omitempty"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Disabled        bool            `json:"disabled"`
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

func (client *APIClient) CreateShiftLeftCveExceptionList(data ShiftLeftCveExceptionList) (*ShiftLeftCveExceptionList, error) {
	resp, err := client.Post("/api/shiftleft/exceptions/", data)
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
