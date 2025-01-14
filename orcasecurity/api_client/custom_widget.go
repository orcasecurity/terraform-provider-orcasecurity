package api_client

import (
	"fmt"
)

type CustomWidgetExtraParametersSettingsField struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

type RequestParams struct {
	Query            map[string]interface{} `json:"query"`
	GroupBy          []string               `json:"group_by"`
	GroupByList      []string               `json:"group_by[],omitempty"`
	AdditionalModels []string               `json:"additional_models[]"`
	Limit            int64                  `json:"limit,omitempty"`
	OrderBy          []string               `json:"order_by[],omitempty"`
	StartAtIndex     int64                  `json:"start_at_index"`
	EnablePagination bool                   `json:"enable_pagination"`
}

type CustomWidgetExtraParametersSettings struct {
	Size              string                                   `json:"size"`
	Columns           []string                                 `json:"columns"`
	Field             CustomWidgetExtraParametersSettingsField `json:"field,omitempty"`
	RequestParameters RequestParams                            `json:"requestParams"`
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
	Settings          []CustomWidgetExtraParametersSettings `json:"settings"`
}

type CustomWidget struct {
	ID                string                      `json:"id"`
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
