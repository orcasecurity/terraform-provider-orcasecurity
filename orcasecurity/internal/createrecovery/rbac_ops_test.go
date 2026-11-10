package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

const (
	stubGroupID = "33333333-3333-3333-3333-333333333333"
	stubRoleID  = "44444444-4444-4444-4444-444444444444"
	stubEmail   = "tf-recovery@example.com"
	stubCloudID = "123456789012"
)

var trustedAccountIDs = map[string]int64{newID: 1, oldID: 2}

func named(id string) string { return fmt.Sprintf(`{"id":%q,"name":%q}`, id, stubTag) }

func namedList(created bool) string { return rows(created, named, dataList) }

func namedData(id string) func(bool) string {
	return func(bool) string { return `{"data":` + named(id) + `}` }
}

func init() {
	opCases = append(opCases,
		opCase{
			name: "business_unit",
			path: "/api/business_units",
			get: map[string]func(bool) string{
				"/api/business_units":          namedList,
				"/api/business_units/" + newID: namedData(newID),
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.BusinessUnitCreateOp(api_client.BusinessUnit{Name: stubTag}))
			},
		},
		opCase{
			name: "custom_role",
			path: "/api/rbac/roles",
			get: map[string]func(bool) string{
				"/api/rbac/role":           namedList,
				"/api/rbac/roles/" + newID: namedData(newID),
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.CustomRoleCreateOp(api_client.CustomRole{Name: stubTag}))
			},
		},
		opCase{
			name:  "group",
			path:  "/api/rbac/group",
			empty: `{"total_items":0,"data":{"groups":[]}}`,
			get: map[string]func(bool) string{
				"/api/rbac/group": func(created bool) string {
					return rows(created, named, func(items string) string { return `{"data":{"groups":[` + items + `]}}` })
				},
				"/api/rbac/group/" + newID: namedData(newID),
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.GroupCreateOp(api_client.Group{Name: stubTag}))
			},
		},
		opCase{
			name: "user_invite",
			path: "/api/user_invites/bulk_create/",
			get: map[string]func(bool) string{
				"/api/user_invites/": func(created bool) string {
					return rows(created, func(id string) string {
						return fmt.Sprintf(`{"id":%q,"email":%q}`, id, stubEmail)
					}, dataList)
				},
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.UserInviteCreateOp(api_client.UserInviteRequest{InviteUserEmails: []string{"TF-Recovery@Example.com"}}))
			},
		},
		opCase{
			name:    "trusted_cloud_account",
			path:    "/api/organization/trusted_accounts",
			adopted: "1",
			get: map[string]func(bool) string{
				"/api/organization/trusted_accounts": func(created bool) string {
					return rows(created, func(id string) string {
						return fmt.Sprintf(`{"id":%d,"cloud_provider":"aws","cloud_provider_id":%q}`, trustedAccountIDs[id], stubCloudID)
					}, dataList)
				},
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.TrustedCloudAccountCreateOp(api_client.TrustedCloudAccount{CloudProvider: "aws", CloudAccountID: stubCloudID}))
			},
		},
		opCase{
			name: "group_access",
			path: "/api/rbac/access/group",
			get: map[string]func(bool) string{
				"/api/rbac/access/group": func(created bool) string {
					return rows(created, func(id string) string {
						return fmt.Sprintf(`{"id":%q,"group":{"id":%q},"role":{"id":%q},"all_cloud_accounts":true}`, id, stubGroupID, stubRoleID)
					}, dataList)
				},
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.GroupAccessCreateOp(api_client.GroupAccess{GroupID: stubGroupID, RoleID: stubRoleID, AllCloudAccounts: true}))
			},
		},
	)
}
