package api_client

import (
	"encoding/json"
	"testing"
)

func TestRbacRoleListAPIResponseUnmarshal(t *testing.T) {
	raw := `{"status":"success","data":[{"id":"a","name":"Viewer"},{"id":"b","name":"Editor"}]}`
	var parsed rbacRoleListAPIResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.Status != "success" || len(parsed.Data) != 2 {
		t.Fatalf("got %+v", parsed)
	}
	if parsed.Data[0].ID != "a" || parsed.Data[0].Name != "Viewer" {
		t.Fatalf("first role %+v", parsed.Data[0])
	}
}
