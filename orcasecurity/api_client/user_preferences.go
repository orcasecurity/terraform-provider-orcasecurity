package api_client

import (
	"fmt"
	"net/url"
)

const CustomWidgetViewType = "customs_widgets"

type UserPreferenceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userPreferencesResponse struct {
	Data struct {
		OrganizationPreferences []UserPreferenceSummary `json:"organization_preferences"`
		UserPreferences         []UserPreferenceSummary `json:"user_preferences"`
	} `json:"data"`
}

// ListUserPreferences returns the organization-level and user-level preferences of one view_type
// (customs_widgets, dashboard, discovery, ...). The endpoint is unpaginated.
func (client *APIClient) ListUserPreferences(viewType string) ([]UserPreferenceSummary, error) {
	resp, err := client.Get("/api/user_preferences?" + url.Values{"view_type": {viewType}}.Encode())
	if err != nil {
		return nil, err
	}

	var parsed userPreferencesResponse
	if err := resp.ReadJSON(&parsed); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var out []UserPreferenceSummary
	for _, rows := range [][]UserPreferenceSummary{parsed.Data.OrganizationPreferences, parsed.Data.UserPreferences} {
		for _, p := range rows {
			if p.ID != "" && !seen[p.ID] {
				seen[p.ID] = true
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// Names are unique per (creating user, name, view_type), so other users' org-level rows can match.
func (client *APIClient) userPreferenceIDsByName(viewType, name string) func() (map[string]bool, error) {
	return func() (map[string]bool, error) {
		all, err := client.ListUserPreferences(viewType)
		if err != nil {
			return nil, err
		}
		return idsWhere(all, func(p *UserPreferenceSummary) string { return p.ID },
			func(p *UserPreferenceSummary) bool { return sameName(p.Name, name) }), nil
	}
}

func (client *APIClient) CustomWidgetCreateOp(data CustomWidget) CreateOp[CustomWidget] {
	return CreateOp[CustomWidget]{
		What:   fmt.Sprintf("custom widget named %q", data.Name),
		ID:     func(o *CustomWidget) string { return o.ID },
		Create: func() (*CustomWidget, error) { return client.createCustomWidget(data) },
		IDs:    client.userPreferenceIDsByName(data.ViewType, data.Name),
		Get:    client.GetCustomWidget,
	}
}

func (client *APIClient) CustomDashboardCreateOp(data CustomDashboard) CreateOp[CustomDashboard] {
	return CreateOp[CustomDashboard]{
		What:   fmt.Sprintf("custom dashboard named %q", data.Name),
		ID:     func(o *CustomDashboard) string { return o.ID },
		Create: func() (*CustomDashboard, error) { return client.createCustomDashboard(data) },
		IDs:    client.userPreferenceIDsByName(data.ViewType, data.Name),
		Get:    client.GetCustomDashboard,
	}
}

func (client *APIClient) DiscoveryViewCreateOp(data DiscoveryView) CreateOp[DiscoveryView] {
	return CreateOp[DiscoveryView]{
		What:   fmt.Sprintf("discovery view named %q", data.Name),
		ID:     func(o *DiscoveryView) string { return o.ID },
		Create: func() (*DiscoveryView, error) { return client.createDiscoveryView(data) },
		IDs:    client.userPreferenceIDsByName(data.ViewType, data.Name),
		Get:    client.GetDiscoveryView,
	}
}
