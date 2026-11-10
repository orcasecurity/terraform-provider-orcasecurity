package api_client

import "fmt"

const apiRBACGroupAccessPath = "/api/rbac/access/group"

var groupAccessEndpoint = rbacAccessEndpoint{
	path:           apiRBACGroupAccessPath,
	ownerKey:       "group",
	paginated:      true,
	noReplayCreate: true,
}

// GroupAccess is the public DTO for a group–role assignment. The shared engine
// in rbac_access.go does the transport; this type just names the group-facing
// fields the resource layer reads.
type GroupAccess struct {
	ID                string
	AllCloudAccounts  bool
	RoleID            string
	GroupID           string
	CloudAccounts     []string
	ShiftleftProjects []string
	UserFilters       []string
}

func (g GroupAccess) toRecord() rbacAccessRecord {
	return rbacAccessRecord{
		ID:                g.ID,
		OwnerID:           g.GroupID,
		RoleID:            g.RoleID,
		AllCloudAccounts:  g.AllCloudAccounts,
		CloudAccounts:     g.CloudAccounts,
		ShiftleftProjects: g.ShiftleftProjects,
		UserFilters:       g.UserFilters,
	}
}

func groupAccessFromRecord(r rbacAccessRecord) GroupAccess {
	return GroupAccess{
		ID:                r.ID,
		GroupID:           r.OwnerID,
		RoleID:            r.RoleID,
		AllCloudAccounts:  r.AllCloudAccounts,
		CloudAccounts:     r.CloudAccounts,
		ShiftleftProjects: r.ShiftleftProjects,
		UserFilters:       r.UserFilters,
	}
}

// createGroupAccess assigns a role to a group with optional cloud account, Shift Left, or user filter (e.g. business unit) scope.
func (client *APIClient) createGroupAccess(data GroupAccess) (*GroupAccess, error) {
	id, err := client.createAccess(groupAccessEndpoint, data.toRecord())
	if err != nil {
		return nil, err
	}
	out := data
	out.ID = id
	return &out, nil
}

// ListGroupAccessForGroup returns the assignments whose nested group id equals groupID.
func (client *APIClient) ListGroupAccessForGroup(groupID string) ([]GroupAccess, error) {
	records, err := client.listAccessForOwner(groupAccessEndpoint, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupAccess, 0, len(records))
	for _, r := range records {
		out = append(out, groupAccessFromRecord(r))
	}
	return out, nil
}

// FindGroupAccess resolves an assignment by id (preferred) or role+scope fallback; nil when nothing matches.
func (client *APIClient) FindGroupAccess(assignmentID string, want GroupAccess) (*GroupAccess, error) {
	rec, err := client.findAccess(groupAccessEndpoint, assignmentID, want.toRecord())
	if err != nil || rec == nil {
		return nil, err
	}
	out := groupAccessFromRecord(*rec)
	return &out, nil
}

// UpdateGroupAccess updates an existing assignment (id carried in the body).
func (client *APIClient) UpdateGroupAccess(data GroupAccess) (*GroupAccess, error) {
	rec, err := client.updateAccess(groupAccessEndpoint, data.toRecord())
	if err != nil {
		return nil, err
	}
	out := groupAccessFromRecord(*rec)
	return &out, nil
}

// DeleteGroupAccess removes a group–role assignment (id carried in the body).
func (client *APIClient) DeleteGroupAccess(id string) error {
	return client.deleteAccess(groupAccessEndpoint, id)
}

func (client *APIClient) ListGroupAccessIDsMatching(want GroupAccess) (map[string]bool, error) {
	return client.accessIDsMatching(groupAccessEndpoint, want.toRecord())
}

func (client *APIClient) GroupAccessCreateOp(data GroupAccess) CreateOp[GroupAccess] {
	return CreateOp[GroupAccess]{
		What:   fmt.Sprintf("access for group %s with role %s and the same scope", data.GroupID, data.RoleID),
		Create: func() (*GroupAccess, error) { return client.createGroupAccess(data) },
		IDs:    func() (map[string]bool, error) { return client.ListGroupAccessIDsMatching(data) },
		Get: func(id string) (*GroupAccess, error) {
			all, err := client.ListGroupAccessForGroup(data.GroupID)
			if err != nil {
				return nil, err
			}
			for i := range all {
				if all[i].ID == id {
					return &all[i], nil
				}
			}
			return nil, nil
		},
	}
}
