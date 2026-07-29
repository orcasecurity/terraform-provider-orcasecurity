package api_client

const apiRBACUserAccessPath = "/api/rbac/access/user"

var userAccessEndpoint = rbacAccessEndpoint{
	path:     apiRBACUserAccessPath,
	ownerKey: "user",
	// The user endpoint is a single unpaginated GET that returns every row and
	// ignores query params — do not send limit/start_at_index.
	paginated: false,
}

// UserAccess is the public DTO for a user–role assignment. The shared engine in
// rbac_access.go does the transport; this type just names the user-facing fields
// the resource layer reads.
type UserAccess struct {
	ID                string
	AllCloudAccounts  bool
	RoleID            string
	UserID            string
	CloudAccounts     []string
	ShiftleftProjects []string
	UserFilters       []string
}

func (u UserAccess) toRecord() rbacAccessRecord {
	return rbacAccessRecord{
		ID:                u.ID,
		OwnerID:           u.UserID,
		RoleID:            u.RoleID,
		AllCloudAccounts:  u.AllCloudAccounts,
		CloudAccounts:     u.CloudAccounts,
		ShiftleftProjects: u.ShiftleftProjects,
		UserFilters:       u.UserFilters,
	}
}

func userAccessFromRecord(r rbacAccessRecord) UserAccess {
	return UserAccess{
		ID:                r.ID,
		UserID:            r.OwnerID,
		RoleID:            r.RoleID,
		AllCloudAccounts:  r.AllCloudAccounts,
		CloudAccounts:     r.CloudAccounts,
		ShiftleftProjects: r.ShiftleftProjects,
		UserFilters:       r.UserFilters,
	}
}

// CreateUserAccess assigns a role to a user with optional cloud account, Shift
// Left, or user filter (e.g. business unit) scope.
func (client *APIClient) CreateUserAccess(data UserAccess) (*UserAccess, error) {
	id, err := client.createAccess(userAccessEndpoint, data.toRecord())
	if err != nil {
		return nil, err
	}
	out := data
	out.ID = id
	return &out, nil
}

// ListUserAccessForUser returns the assignments whose nested user id equals userID.
func (client *APIClient) ListUserAccessForUser(userID string) ([]UserAccess, error) {
	records, err := client.listAccessForOwner(userAccessEndpoint, userID)
	if err != nil {
		return nil, err
	}
	out := make([]UserAccess, 0, len(records))
	for _, r := range records {
		out = append(out, userAccessFromRecord(r))
	}
	return out, nil
}

// FindUserAccess resolves an assignment by id (preferred) or role+scope fallback; nil when nothing matches.
func (client *APIClient) FindUserAccess(assignmentID string, want UserAccess) (*UserAccess, error) {
	rec, err := client.findAccess(userAccessEndpoint, assignmentID, want.toRecord())
	if err != nil || rec == nil {
		return nil, err
	}
	out := userAccessFromRecord(*rec)
	return &out, nil
}

// UpdateUserAccess updates an existing assignment (id carried in the body).
func (client *APIClient) UpdateUserAccess(data UserAccess) (*UserAccess, error) {
	rec, err := client.updateAccess(userAccessEndpoint, data.toRecord())
	if err != nil {
		return nil, err
	}
	out := userAccessFromRecord(*rec)
	return &out, nil
}

// DeleteUserAccess removes a user–role assignment (id carried in the body).
func (client *APIClient) DeleteUserAccess(id string) error {
	return client.deleteAccess(userAccessEndpoint, id)
}
