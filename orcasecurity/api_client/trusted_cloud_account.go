package api_client

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TrustedCloudAccount struct {
	ID             int64  `json:"id,omitempty"`
	Name           string `json:"account_name"`
	Description    string `json:"description"`
	CloudProvider  string `json:"cloud_provider,omitempty"`
	Provider       string `json:"provider,omitempty"`
	CloudAccountID string `json:"cloud_provider_id"`
}

type trustedCloudAccountAPIResponseType struct {
	Data   TrustedCloudAccount `json:"data"`
	Status string              `json:"status"`
}

type trustedCloudAccountAPIResponseTypeRead struct {
	Data   []TrustedCloudAccount `json:"data"`
	Status string                `json:"status"`
}

func (client *APIClient) DoesTrustedCloudAccountExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/organization/trusted_accounts?id=%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetTrustedCloudAccount(id string) (*TrustedCloudAccount, error) {
	resp, err := client.Get(fmt.Sprintf("/api/organization/trusted_accounts?id=%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := trustedCloudAccountAPIResponseTypeRead{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data[0], nil
}

func (client *APIClient) CreateTrustedCloudAccount(data TrustedCloudAccount) (*TrustedCloudAccount, error) {
	resp, err := client.Post("/api/organization/trusted_accounts", data)
	if err != nil {
		return nil, err
	}

	response := trustedCloudAccountAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateTrustedCloudAccount(data TrustedCloudAccount) (*TrustedCloudAccount, error) {
	resp, err := client.Put(fmt.Sprintf("/api/organization/trusted_accounts?id=%s", strconv.Itoa(int(data.ID))), data)
	if err != nil {
		return nil, err
	}

	response := trustedCloudAccountAPIResponseType{}
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteTrustedCloudAccount(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/organization/trusted_accounts?id=%s", id))
	return err
}
