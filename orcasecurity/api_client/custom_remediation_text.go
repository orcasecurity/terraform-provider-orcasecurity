package api_client

import (
	"errors"
	"net/http"
)

const customRemediationTextPath = "/api/alerts/custom_remediation_text"

// The POST is a create guarded by unique(organization, alert_type): a replay of a committed POST gets
// 400 "already set". After an ambiguous POST, the PUT (set-to-value) either updates the committed row
// or 404s because nothing was created, in which case POSTing again is safe.
func (client *APIClient) setCustomRemediationText(data any) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		resp, putErr := client.Put(customRemediationTextPath, data)
		if resp == nil || resp.StatusCode() != http.StatusNotFound {
			return putErr
		}
		if _, err = client.Post(customRemediationTextPath, data); !errors.Is(err, ErrAmbiguousWrite) {
			return err
		}
	}
	return err
}
