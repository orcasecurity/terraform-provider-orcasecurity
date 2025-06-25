package api_client

import (
	"fmt"
	"net/url"
)

type user struct {
	ID       string   `json:"user_id"`
	Email    string   `json:"email"`
	First    string   `json:"first"`
	Last     string   `json:"last"`
	MFARequired bool  `json:"mfa_required"`
	MFAEnabled bool   `json:"mfa_enabled"`
}

func (client *APIClient) GetUserByEmail(email string) (*user, error) {
	resp, err := client.Get(
		fmt.Sprintf("/api/users?search=%s&limit=10",
			url.QueryEscape(email),
		),
	)
	if err != nil {
		return nil, err
	}
	type respsoneType struct {
		Data []user `json:"data"`
	}
	response := respsoneType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("user with email '%s' does not exists", email)
	}
	if len(response.Data) > 1 {
		return nil, fmt.Errorf("too many results for user with email '%s'. expected one", email)
	}
	return &response.Data[0], nil
}
	