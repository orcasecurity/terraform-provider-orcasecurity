package api_client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UserInvite maps to the /api/user_invites endpoints ("Add Users" in the UI).
// Invites are created via the bulk_create action, listed via the collection, and
// removed via DELETE /api/user_invites/<id>. There is no update endpoint.
const (
	apiUserInvitesBulkCreatePath = "/api/user_invites/bulk_create/"
	apiUserInvitesListPath       = "/api/user_invites/"
	// Trailing slash is required: the DRF router redirects the slash-less URL to
	// the canonical form, and the redirect downgrades DELETE to GET (405).
	apiUserInviteByIDFmt = "/api/user_invites/%s/"
)

// UserInviteRequest is the bulk_create payload. For a single-invite resource the
// email is sent as a one-element list.
type UserInviteRequest struct {
	InviteUserEmails  []string `json:"invite_user_emails"`
	RoleID            string   `json:"role_id,omitempty"`
	Groups            []string `json:"groups"`
	AllCloudAccounts  bool     `json:"all_cloud_accounts"`
	CloudAccounts     []string `json:"cloud_accounts"`
	UserFilters       []string `json:"user_filters"`
	ShiftleftProjects []string `json:"shiftleft_projects"`
	MFARequired       bool     `json:"mfa_required"`
	ShouldSendEmail   bool     `json:"should_send_email"`
}

// UserInvite is the API representation of a pending invite.
type UserInvite struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	InviteLink string `json:"invite_link"`
	Expired    bool   `json:"expired"`
}

// CreateUserInvite invites a single user and returns the created invite.
func (client *APIClient) CreateUserInvite(req UserInviteRequest) (*UserInvite, error) {
	resp, err := client.Post(apiUserInvitesBulkCreatePath, req)
	if err != nil {
		return nil, err
	}

	// bulk_create returns UserInviteSerializer(many=True). Tolerate a bare list
	// as well as the {"data": [...]} envelope used elsewhere.
	invites, err := parseUserInviteList(resp.Body())
	if err != nil {
		return nil, err
	}
	if len(invites) == 0 {
		return nil, fmt.Errorf("create user invite: no invite returned in response: %s", string(resp.Body()))
	}
	return &invites[0], nil
}

func parseUserInviteList(body []byte) ([]UserInvite, error) {
	// Bare list: [ {...} ]
	var bare []UserInvite
	if err := json.Unmarshal(body, &bare); err == nil && len(bare) > 0 {
		return bare, nil
	}
	// Envelopes: {"data": [...]} or paginated {"results": [...]}.
	var envelope struct {
		Data    []UserInvite `json:"data"`
		Results []UserInvite `json:"results"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Data) > 0 {
		return envelope.Data, nil
	}
	return envelope.Results, nil
}

// ListUserInvites returns all pending invites in the organization.
func (client *APIClient) ListUserInvites() ([]UserInvite, error) {
	resp, err := client.Get(apiUserInvitesListPath)
	if err != nil {
		return nil, err
	}
	return parseUserInviteList(resp.Body())
}

// GetUserInvite finds a pending invite by id, returning nil when it no longer
// exists (e.g. the invitee registered or the invite was revoked).
func (client *APIClient) GetUserInvite(id string) (*UserInvite, error) {
	invites, err := client.ListUserInvites()
	if err != nil {
		return nil, err
	}
	for _, inv := range invites {
		if inv.ID == id {
			found := inv
			return &found, nil
		}
	}
	return nil, nil
}

// DeleteUserInvite revokes a pending invite.
func (client *APIClient) DeleteUserInvite(id string) error {
	_, err := client.Delete(fmt.Sprintf(apiUserInviteByIDFmt, id))
	if err != nil && strings.Contains(err.Error(), "status: 404") {
		return nil
	}
	return err
}
