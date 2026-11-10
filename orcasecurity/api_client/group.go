package api_client

import (
	"fmt"
	"strings"
)

type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SSOGroup    bool   `json:"sso_group"`
	// omitempty: update fully replaces membership via this field, but only when it is
	// non-empty, and never clears it to zero. Callers manage membership explicitly via
	// AddGroupUsers/RemoveGroupUsers instead of relying on that side effect.
	Users []string `json:"users,omitempty"`
}

type groupAPIResponseType struct {
	Data Group `json:"data"`
}

func (client *APIClient) DoesGroupExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/rbac/group/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetGroup(id string) (*Group, error) {
	resp, err := client.Get(fmt.Sprintf("/api/rbac/group/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	group := response.Data
	group.ID = id
	return &group, nil
}

func (client *APIClient) createGroup(data Group) (*Group, error) {
	resp, err := client.PostCreate("/api/rbac/group", data)
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) UpdateGroup(data Group) (*Group, error) {
	resp, err := client.Put(fmt.Sprintf("/api/rbac/group/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := groupAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteGroup(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/rbac/group/%s", id))
	return err
}

// Create ignores users; membership is set via POST /api/rbac/group/{id}/users (no-op when empty).
func (client *APIClient) AddGroupUsers(groupID string, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := client.Post(
		fmt.Sprintf("/api/rbac/group/%s/users", groupID),
		map[string][]string{"user_ids": userIDs},
	)
	return err
}

// RemoveGroupUsers removes members via DELETE /api/rbac/group/{id}/users (no-op when empty).
func (client *APIClient) RemoveGroupUsers(groupID string, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	_, err := client.DeleteWithBody(
		fmt.Sprintf("/api/rbac/group/%s/users", groupID),
		map[string][]string{"user_ids": userIDs},
	)
	return err
}

type groupListRow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// The list nests rows one level down: {"total_items": n, "data": {"groups": [...]}}.
type groupListPage struct {
	Data *struct {
		Groups *[]groupListRow `json:"groups"`
	} `json:"data"`
}

func (client *APIClient) ListGroupIDsByName(name string) (map[string]bool, error) {
	all, err := paginateOffsetWith(client, "/api/rbac/group", listFilters{"search": strings.TrimSpace(name)}, 300, 500,
		func(resp *APIResponse) (*[]groupListRow, error) {
			var page groupListPage
			if err := resp.ReadJSON(&page); err != nil || page.Data == nil {
				return nil, err
			}
			return page.Data.Groups, nil
		})
	if err != nil {
		return nil, err
	}
	return idsWhere(all, func(g *groupListRow) string { return g.ID }, func(g *groupListRow) bool { return sameName(g.Name, name) }), nil
}

func (client *APIClient) GroupCreateOp(data Group) CreateOp[Group] {
	return CreateOp[Group]{
		What:   fmt.Sprintf("group named %q", data.Name),
		Create: func() (*Group, error) { return client.createGroup(data) },
		IDs:    func() (map[string]bool, error) { return client.ListGroupIDsByName(data.Name) },
		Get:    client.GetGroup,
	}
}
