package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func init() {
	opCases = append(opCases,
		opCase{
			name: "business_unit",
			post: "/api/business_units",
			get: map[string]func(bool) string{
				"/api/business_units": func(created bool) string {
					return rows(created, func(id string) string {
						return fmt.Sprintf(`{"id":%q,"name":%q}`, id, stubTag)
					}, dataList)
				},
				"/api/business_units/" + newID: func(bool) string {
					return fmt.Sprintf(`{"data":{"id":%q,"name":%q}}`, newID, stubTag)
				},
			},
			op: func(c *api_client.APIClient) runner {
				return runOp(c.BusinessUnitCreateOp(api_client.BusinessUnit{Name: stubTag}),
					func(b *api_client.BusinessUnit) string { return b.ID })
			},
		},
	)
}
