package api_client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const deletedRuleRecheckDelay = 1 * time.Second

// getSonarRule fetches a custom sonar rule. The API has no 404 here: unknown ids
// return 400 and deleted rules 500, and that 500 is the same generic page as a
// real server error. A 500 is re-checked once so a transient failure during
// refresh is not mistaken for a deleted alert and dropped from state.
func (client *APIClient) getSonarRule(id string) (resp *APIResponse, missing bool, err error) {
	path := fmt.Sprintf("/api/sonar/rules/%s", id)
	resp, err = client.Get(path)
	if resp != nil && resp.StatusCode() == http.StatusInternalServerError {
		if sleepErr := retrySleep(context.Background(), deletedRuleRecheckDelay+retryJitter(deletedRuleRecheckDelay/2)); sleepErr != nil {
			return nil, false, sleepErr
		}
		resp, err = client.Get(path)
	}
	if resp != nil && (resp.StatusCode() == http.StatusBadRequest || resp.StatusCode() == http.StatusInternalServerError) {
		return resp, true, nil
	}
	return resp, false, err
}
