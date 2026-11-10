package api_client

import (
	"errors"
	"fmt"
	"net/http"
)

const customRemediationTextPath = "/api/alerts/custom_remediation_text"

// The POST is a create guarded by unique(organization, alert_type): a replay of a committed POST gets
// 400 "already set". After an ambiguous POST, the PUT (set-to-value) either updates the committed row
// or 404s because nothing was created, in which case POSTing again is safe. A 504'd POST may still be
// saving when that PUT runs, so a 400 on the follow-up POST means the row now exists: PUT again.
func (client *APIClient) setCustomRemediationText(data any) error {
	var err error
	ambiguous := false
	for attempt := 0; attempt < 3; attempt++ {
		resp, putErr := client.Put(customRemediationTextPath, data)
		if resp == nil || resp.StatusCode() != http.StatusNotFound {
			return putErr
		}
		resp, err = client.postWithResponse(noReplayContext(), customRemediationTextPath, data)
		switch {
		case errors.Is(err, ErrAmbiguousWrite):
			ambiguous = true
		case ambiguous && resp != nil && resp.StatusCode() == http.StatusBadRequest:
		default:
			return err
		}
	}
	return err
}

// rollbackCreatedRule deletes a rule created in the same Create whose remediation text failed:
// Terraform records no state on a failed Create, so a surviving rule would be left untracked.
func rollbackCreatedRule(textErr error, ruleID string, deleteRule func(string) error) error {
	if ruleID == "" {
		return fmt.Errorf("remediation text create failed: %w", textErr)
	}
	if delErr := deleteRule(ruleID); delErr != nil {
		return fmt.Errorf("remediation text create failed: %w; deleting the new rule %s also failed (%v), so delete or import it before re-applying",
			textErr, ruleID, delErr)
	}
	return fmt.Errorf("remediation text create failed, so the new rule %s was deleted: %w", ruleID, textErr)
}
