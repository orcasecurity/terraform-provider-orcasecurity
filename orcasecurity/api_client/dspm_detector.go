package api_client

import (
	"fmt"
	"net/url"
)

const dspmDetectorBasePath = "/api/scan_configuration/dspm_detector"

// DSPMDetectorCondition is one content-matching condition of a detector.
type DSPMDetectorCondition struct {
	Source   string `json:"source"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// DSPMDetectorProperties holds the detection configuration of a detector.
type DSPMDetectorProperties struct {
	Conditions      []DSPMDetectorCondition `json:"conditions"`
	DetectionTypes  []string                `json:"detection_types,omitempty"`
	Sensitivity     string                  `json:"sensitivity,omitempty"`
	Significance    string                  `json:"significance,omitempty"`
	Keywords        []string                `json:"keywords,omitempty"`
	ExcludeKeywords []string                `json:"exclude_keywords,omitempty"`
	StopWildcards   []string                `json:"stop_wildcards,omitempty"`
	TextThreshold   *int64                  `json:"text_threshold,omitempty"`
	DBThreshold     *int64                  `json:"db_threshold,omitempty"`
	OCRThreshold    *int64                  `json:"ocr_threshold,omitempty"`
	AIThreshold     *int64                  `json:"ai_threshold,omitempty"`
}

// DSPMDetector is a DSPM detector ("Sensitive Data Identifier" in the UI)
// from /api/scan_configuration/dspm_detector.
type DSPMDetector struct {
	ID             string                 `json:"id,omitempty"`
	OrganizationID string                 `json:"organization,omitempty"`
	Title          string                 `json:"title"`
	Details        string                 `json:"details"`
	Category       string                 `json:"category"`
	SubCategory    string                 `json:"sub_category"`
	IsDisabled     bool                   `json:"is_disabled"`
	IsCustom       bool                   `json:"is_custom"`
	Properties     DSPMDetectorProperties `json:"properties"`
}

// DSPMDetectorListFilters are optional query filters for ListDSPMDetectors.
type DSPMDetectorListFilters struct {
	Title       string
	Category    string
	SubCategory string
}

// readDSPMDetectorData decodes the response envelope strictly: a payload that
// does not carry a detector id is a decode-shape mismatch, not a valid detector.
func readDSPMDetectorData(resp *APIResponse) (*DSPMDetector, error) {
	detector, err := readData[DSPMDetector](resp)
	if err != nil {
		return nil, err
	}
	if detector.ID == "" {
		return nil, fmt.Errorf("dspm detector: could not decode response: %s", string(resp.Body()))
	}
	return detector, nil
}

// GetDSPMDetector retrieves one detector. Returns (nil, nil) on 404 so the
// resource Read can RemoveResource on remote drift.
func (client *APIClient) GetDSPMDetector(id string) (*DSPMDetector, error) {
	resp, err := client.Get(fmt.Sprintf("%s/%s", dspmDetectorBasePath, id))
	if resp != nil && resp.StatusCode() == 404 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return readDSPMDetectorData(resp)
}

// ListDSPMDetectors lists detectors, optionally filtered by title/category/sub_category.
func (client *APIClient) ListDSPMDetectors(filters DSPMDetectorListFilters) ([]DSPMDetector, error) {
	query := url.Values{}
	if filters.Title != "" {
		query.Set("title", filters.Title)
	}
	if filters.Category != "" {
		query.Set("category", filters.Category)
	}
	if filters.SubCategory != "" {
		query.Set("sub_category", filters.SubCategory)
	}
	path := dspmDetectorBasePath
	if len(query) > 0 {
		path = fmt.Sprintf("%s?%s", path, query.Encode())
	}

	resp, err := client.Get(path)
	if err != nil {
		return nil, err
	}
	detectors, err := readData[[]DSPMDetector](resp)
	if err != nil {
		return nil, err
	}
	return *detectors, nil
}

func (client *APIClient) CreateDSPMDetector(data DSPMDetector) (*DSPMDetector, error) {
	resp, err := client.Post(dspmDetectorBasePath, data)
	if err != nil {
		return nil, err
	}
	return readDSPMDetectorData(resp)
}

func (client *APIClient) UpdateDSPMDetector(id string, data DSPMDetector) (*DSPMDetector, error) {
	resp, err := client.Put(fmt.Sprintf("%s/%s", dspmDetectorBasePath, id), data)
	if err != nil {
		return nil, err
	}
	return readDSPMDetectorData(resp)
}

func (client *APIClient) DeleteDSPMDetector(id string) error {
	_, err := client.Delete(fmt.Sprintf("%s/%s", dspmDetectorBasePath, id))
	return err
}
