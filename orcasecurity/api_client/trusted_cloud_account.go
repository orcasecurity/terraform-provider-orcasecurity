package api_client

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	for i := range response.Data {
		if strconv.FormatInt(response.Data[i].ID, 10) == id {
			return &response.Data[i], nil
		}
	}
	return nil, nil
}

func (client *APIClient) createTrustedCloudAccount(data TrustedCloudAccount) (*TrustedCloudAccount, error) {
	resp, err := client.PostCreate("/api/organization/trusted_accounts", data)
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

// Query params feed the ORM filter as-is (cloud_provider is an int column there), so filter on cloud_provider_id only.
func (client *APIClient) ListTrustedCloudAccountIDs(cloudProvider, cloudAccountID string) (map[string]bool, error) {
	resp, err := client.Get("/api/organization/trusted_accounts?" + url.Values{"cloud_provider_id": {cloudAccountID}}.Encode())
	if err != nil {
		return nil, err
	}
	var response trustedCloudAccountAPIResponseTypeRead
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return nil, err
	}
	return idsWhere(response.Data, trustedCloudAccountID, func(a *TrustedCloudAccount) bool {
		return a.CloudProvider == cloudProvider && a.CloudAccountID == cloudAccountID
	}), nil
}

func trustedCloudAccountID(a *TrustedCloudAccount) string {
	if a.ID == 0 {
		return ""
	}
	return strconv.FormatInt(a.ID, 10)
}

func (client *APIClient) TrustedCloudAccountCreateOp(data TrustedCloudAccount) CreateOp[TrustedCloudAccount] {
	return CreateOp[TrustedCloudAccount]{
		What:   fmt.Sprintf("trusted %s cloud account %q", data.CloudProvider, data.CloudAccountID),
		ID:     trustedCloudAccountID,
		Create: func() (*TrustedCloudAccount, error) { return client.createTrustedCloudAccount(data) },
		IDs: func() (map[string]bool, error) {
			return client.ListTrustedCloudAccountIDs(data.CloudProvider, data.CloudAccountID)
		},
		Get: client.GetTrustedCloudAccount,
	}
}
