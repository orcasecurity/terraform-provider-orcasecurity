package api_client

import (
	"fmt"
)

type CustomWidgetExtraParametersSettingsField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type CustomWidgetExtraParametersSettingsRequestParametersQuery struct {
	Query            map[string]interface{} `json:"query"`
	AdditionalModels []string               `json:"additional_models[]"`
	GroupBy          []string               `json:"group_by"`
	GroupByArray     []string               `json:"group_by[]"`
}

type CustomWidgetExtraParametersSettings struct {
	Size              string                                   `json:"size"`
	Field             CustomWidgetExtraParametersSettingsField `json:"field"`
	RequestParameters string                                   `json:"requestParams"`
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
	ID                string                      `json:"preference_id"`
	Name              string                      `json:"name"`
	FilterData        map[string]interface{}      `json:"filter_data"`
	OrganizationLevel bool                        `json:"organization_level"`
	ViewType          string                      `json:"view_type"`
	ExtraParameters   CustomWidgetExtraParameters `json:"extra_params"`
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

	response := customWidgetAPIResponseType{}
	if err = resp.ReadJSON(&response); err != nil {
		return nil, err
	}

	customWidget := response.Data
	customWidget.ID = id
	return &customWidget, nil
}

func (client *APIClient) CreateCustomWidget(data CustomWidget) (*CustomWidget, error) {
	resp, err := client.Post("/api/user_preferences", data)
	if err != nil {
		return nil, err
	}

	response := customWidgetAPIResponseType{}
	err = resp.ReadJSON(&response)
	if err != nil {
		return nil, err
	}

	//customWidget := response.Data
	//customWidget.ID = response.Data.ID
	return &response.Data, nil
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
