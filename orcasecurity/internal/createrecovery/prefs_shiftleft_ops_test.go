package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func prefsList(items string) string {
	return `{"data":{"organization_preferences":[],"user_preferences":[` + items + `]}}`
}

func userPreferenceCase(name, viewType string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: "/api/user_preferences",
		get: map[string]func(bool) string{
			"/api/user_preferences": func(created bool) string {
				return rows(created, func(id string) string {
					return fmt.Sprintf(`{"id":%q,"name":%q,"organization_level":false}`, id, stubTag)
				}, prefsList)
			},
			"/api/user_preferences/" + newID: func(bool) string {
				return fmt.Sprintf(`{"data":{"id":%q,"name":%q,"view_type":%q}}`, newID, stubTag, viewType)
			},
		},
		op: op,
	}
}

func shiftLeftCase(name, path, row string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: path,
		get: map[string]func(bool) string{
			path: func(created bool) string {
				return rows(created, func(id string) string { return fmt.Sprintf(row, id) }, dataList)
			},
			path + newID + "/": func(bool) string { return fmt.Sprintf(row, newID) },
		},
		op: op,
	}
}

func init() {
	opCases = append(opCases,
		userPreferenceCase("custom_dashboard", "dashboard", func(c *api_client.APIClient) runner {
			return runOp(c.CustomDashboardCreateOp(api_client.CustomDashboard{Name: stubTag, ViewType: "dashboard"}),
				func(d *api_client.CustomDashboard) string { return d.ID })
		}),
		userPreferenceCase("custom_widget", api_client.CustomWidgetViewType, func(c *api_client.APIClient) runner {
			return runOp(c.CustomWidgetCreateOp(api_client.CustomWidget{Name: stubTag, ViewType: api_client.CustomWidgetViewType}),
				func(w *api_client.CustomWidget) string { return w.ID })
		}),
		userPreferenceCase("discovery_view", "discovery", func(c *api_client.APIClient) runner {
			return runOp(c.DiscoveryViewCreateOp(api_client.DiscoveryView{Name: stubTag, ViewType: "discovery"}),
				func(v *api_client.DiscoveryView) string { return v.ID })
		}),
		shiftLeftCase("shift_left_project", "/api/shiftleft/projects/",
			`{"id":%q,"name":"`+stubTag+`","key":"`+stubTag+`"}`,
			func(c *api_client.APIClient) runner {
				return runOp(c.ShiftLeftProjectCreateOp(api_client.ShiftLeftProject{Name: stubTag, Key: stubTag}),
					func(p *api_client.ShiftLeftProject) string { return p.ID })
			}),
		shiftLeftCase("shift_left_cve_exception_list", "/api/shiftleft/exceptions/",
			`{"id":%q,"name":"`+stubTag+`"}`,
			func(c *api_client.APIClient) runner {
				return runOp(c.ShiftLeftCveExceptionListCreateOp(api_client.ShiftLeftCveExceptionList{Name: stubTag}),
					func(e *api_client.ShiftLeftCveExceptionList) string { return e.ID })
			}),
	)
}
