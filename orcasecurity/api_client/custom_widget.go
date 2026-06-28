package api_client

import (
	"encoding/json"
	"fmt"
)

type CustomWidgetExtraParametersSettingsField struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type RequestParams struct {
	Query            map[string]interface{} `json:"query"`
	GroupBy          []string               `json:"group_by"`
	GroupByBracket   []string               `json:"group_by[],omitempty"`
	GroupByList      []string               `json:"group_by_list,omitempty"`
	AdditionalModels []string               `json:"additional_models[]"`
	Limit            int64                  `json:"limit,omitempty"`
	OrderBy          []string               `json:"order_by[],omitempty"`
	StartAtIndex     *int64                 `json:"start_at_index,omitempty"`
	EnablePagination *bool                  `json:"enable_pagination,omitempty"`
}

// ComparisonRequestParam is one entry in a PIE_CHART_MULTI request params array.
type ComparisonRequestParam struct {
	ID     string        `json:"id"`
	Title  string        `json:"title"`
	Params RequestParams `json:"params"`
}

// WidgetInnerExtraParams is the extra "extraParams" sub-object used by PIE_CHART_MULTI
// (and possibly other compound widget types) — sibling of settings/requestParams.
type WidgetInnerExtraParams struct {
	Field         *CustomWidgetExtraParametersSettingsField `json:"field,omitempty"`
	ValuesFormat  string                                    `json:"valuesFormat,omitempty"`
	DefaultMapper map[string]interface{}                    `json:"defaultMapper,omitempty"`
}

// CustomWidgetExtraParametersSettings holds widget settings. V1 API uses requestParams;
// V2 API uses requestParams2 in the response. Both are supported for Read/Import.
// RequestParams2 is polymorphic: an object for single-query widgets, an array of
// ComparisonRequestParam for PIE_CHART_MULTI — modeled as json.RawMessage.
type CustomWidgetExtraParametersSettings struct {
	Size              string                                    `json:"size"`
	Columns           []string                                  `json:"columns,omitempty"`
	Field             *CustomWidgetExtraParametersSettingsField `json:"field,omitempty"`
	ExtraParams       *WidgetInnerExtraParams                   `json:"extraParams,omitempty"`
	RequestParameters *RequestParams                            `json:"requestParams,omitempty"`
	RequestParams2    json.RawMessage                           `json:"requestParams2,omitempty"`
}

type CustomWidgetExtraParameters struct {
	Type              string                                `json:"type"`
	Category          string                                `json:"category"`
	EmptyStateMessage string                                `json:"emptyStateMessage"`
	Size              string                                `json:"size"`
	IsNew             bool                                  `json:"isNew"`
	Title             string                                `json:"title"`
	Subtitle          string                                `json:"subtitle"`
	Description       string                                `json:"description"`
	ExtraParams       *WidgetInnerExtraParams               `json:"extraParams,omitempty"`
	RequestParams     json.RawMessage                       `json:"requestParams,omitempty"`
	Settings          []CustomWidgetExtraParametersSettings `json:"settings"`
}

type CustomWidget struct {
	ID                string                      `json:"id,omitempty"`
	Name              string                      `json:"name"`
	FilterData        map[string]interface{}      `json:"filter_data"`
	OrganizationLevel bool                        `json:"organization_level"`
	ViewType          string                      `json:"view_type"`
	ExtraParameters   CustomWidgetExtraParameters `json:"extra_params"`
}

// Struct for Create/Update API responses
type customWidgetCreateResponse struct {
	Data struct {
		PreferenceID      string                      `json:"preference_id"`
		Name              string                      `json:"name"`
		FilterData        map[string]interface{}      `json:"filter_data"`
		OrganizationLevel bool                        `json:"organization_level"`
		ViewType          string                      `json:"view_type"`
		ExtraParameters   CustomWidgetExtraParameters `json:"extra_params"`
	} `json:"data"`
}

// Struct for Read API responses
type customWidgetReadResponse struct {
	Data struct {
		ID                string                      `json:"id"`
		Name              string                      `json:"name"`
		FilterData        map[string]interface{}      `json:"filter_data"`
		OrganizationLevel bool                        `json:"organization_level"`
		ViewType          string                      `json:"view_type"`
		ExtraParameters   CustomWidgetExtraParameters `json:"extra_params"`
	} `json:"data"`
}

type customWidgetAPIResponseType struct {
	Data CustomWidget `json:"data"`
}

func (client *APIClient) DoesCustomWidgetExist(id string) (bool, error) {
	resp, _ := client.Head(fmt.Sprintf("/api/user_preferences/%s", id))
	return resp.StatusCode() == 200, nil
}

func (client *APIClient) GetCustomWidget(id string) (*CustomWidget, error) {
	resp, err := client.Get(fmt.Sprintf("/api/user_preferences/%s", id))
	if resp.StatusCode() == 400 || resp.StatusCode() == 500 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	response := customWidgetReadResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	// Map the response to your internal struct
	return &CustomWidget{
		ID:                response.Data.ID,
		Name:              response.Data.Name,
		FilterData:        response.Data.FilterData,
		OrganizationLevel: response.Data.OrganizationLevel,
		ViewType:          response.Data.ViewType,
		ExtraParameters:   response.Data.ExtraParameters,
	}, nil
}

func (client *APIClient) CreateCustomWidget(data CustomWidget) (*CustomWidget, error) {
	resp, err := client.Post("/api/user_preferences", data)
	if err != nil {
		return nil, err
	}

	response := customWidgetCreateResponse{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	return &CustomWidget{
		ID:                response.Data.PreferenceID,
		Name:              response.Data.Name,
		FilterData:        response.Data.FilterData,
		OrganizationLevel: response.Data.OrganizationLevel,
		ViewType:          response.Data.ViewType,
		ExtraParameters:   response.Data.ExtraParameters,
	}, nil
}

func (client *APIClient) UpdateCustomWidget(data CustomWidget) (*CustomWidget, error) {
	resp, err := client.Put(fmt.Sprintf("/api/user_preferences/%s", data.ID), data)
	if err != nil {
		return nil, err
	}

	response := customWidgetAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

func (client *APIClient) DeleteCustomWidget(id string) error {
	_, err := client.Delete(fmt.Sprintf("/api/user_preferences/%s", id))
	return err
}

// CustomWidgetSummary is a lightweight widget representation for listing (id, name only).
type CustomWidgetSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userPreferencesResponse struct {
	Data struct {
		OrganizationPreferences []CustomWidgetSummary `json:"organization_preferences"`
		UserPreferences         []CustomWidgetSummary `json:"user_preferences"`
	} `json:"data"`
}

// ListCustomWidgets fetches custom widget IDs and names via GET /api/user_preferences?view_type=customs_widgets.
// Returns both organization-level and user-level custom widgets.
func (client *APIClient) ListCustomWidgets() ([]CustomWidgetSummary, error) {
	resp, err := client.Get("/api/user_preferences?view_type=customs_widgets")
	if err != nil {
		return nil, err
	}

	var parsed userPreferencesResponse
	if err := resp.ReadJSON(&parsed); err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var out []CustomWidgetSummary
	for _, w := range parsed.Data.OrganizationPreferences {
		if w.ID != "" && !seen[w.ID] {
			seen[w.ID] = true
			out = append(out, w)
		}
	}
	for _, w := range parsed.Data.UserPreferences {
		if w.ID != "" && !seen[w.ID] {
			seen[w.ID] = true
			out = append(out, w)
		}
	}
	return out, nil
}
