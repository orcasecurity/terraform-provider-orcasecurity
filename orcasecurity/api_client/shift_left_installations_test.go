package api_client

import (
	"encoding/json"
	"testing"
)

func TestGitlabInstallationWrite_AlwaysSendsReadOnly(t *testing.T) {
	raw, _ := json.Marshal(GitlabInstallationWrite{Name: "n", AccessToken: "tok"})
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	if _, ok := got["read_only"]; !ok {
		t.Errorf("read_only must always be present: %s", raw)
	}
	if _, ok := got["server_url"]; ok {
		t.Errorf("empty server_url must be omitted: %s", raw)
	}
}

func TestGitlabInstallation_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/gitlab/installations/ (redacted).
	fixture := `{"id":"11111111-1111-1111-1111-111111111111","server_url":"https://gitlab.com","external_server_url":null,"access_token_name":"tok-name","access_token_type":"group","read_only":true,"name":"My GitLab","integration_status":null,"cloud_integration":true,"created_at":"2026-01-01T00:00:00Z","created_by":"u@example.com"}`
	var inst GitlabInstallation
	if err := json.Unmarshal([]byte(fixture), &inst); err != nil {
		t.Fatal(err)
	}
	if !inst.ReadOnly || !inst.CloudIntegration || inst.AccessTokenName != "tok-name" {
		t.Errorf("bad unmarshal: %+v", inst)
	}
}

func TestBitbucketInstallationWrite_NestedTokenDetails(t *testing.T) {
	raw, _ := json.Marshal(BitbucketInstallationWrite{
		Name: "n",
		AccessTokenDetails: &BitbucketAccessTokenDetails{
			AccessToken:     "tok",
			AccessTokenType: "TOKEN",
			AccountID:       "workspace-slug",
		},
	})
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	details := got["access_token_details"].(map[string]any)
	if details["access_token"] != "tok" || details["access_token_type"] != "TOKEN" || details["account_id"] != "workspace-slug" {
		t.Errorf("bad access_token_details: %v", details)
	}
	if _, ok := details["username"]; ok {
		t.Errorf("empty username must be omitted: %v", details)
	}
}

func TestBitbucketInstallation_UnmarshalResponseWithoutSecret(t *testing.T) {
	// The API never echoes access_token; the details block carries metadata only.
	fixture := `{"id":"22222222-2222-2222-2222-222222222222","name":"BB","server_url":"https://bitbucket.org","external_server_url":null,"access_token_details":{"username":null,"access_token_type":"TOKEN","account_id":"ws"},"cloud_integration":true,"integration_status":null,"created_at":"2026-01-01T00:00:00Z","created_by":"u"}`
	var inst BitbucketInstallation
	if err := json.Unmarshal([]byte(fixture), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.AccessTokenDetails == nil || inst.AccessTokenDetails.AccountID != "ws" || inst.AccessTokenDetails.AccessToken != "" {
		t.Errorf("bad token details: %+v", inst.AccessTokenDetails)
	}
}

func TestAzureDevopsInstallationWrite_TokenDetails(t *testing.T) {
	raw, _ := json.Marshal(AzureDevopsInstallationWrite{
		Name: "n",
		AccessTokenDetails: &AzureAccessTokenDetails{
			AccessToken: "tok",
			AccountName: "org",
		},
	})
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	details := got["access_token_details"].(map[string]any)
	if details["access_token"] != "tok" || details["account_name"] != "org" {
		t.Errorf("bad access_token_details: %v", details)
	}
}

func TestAzureDevopsInstallation_UnmarshalLiveShape(t *testing.T) {
	// Fixture captured from GET /api/shiftleft/azure_devops/installations/ (redacted).
	fixture := `{"id":"33333333-3333-3333-3333-333333333333","server_url":"https://dev.azure.com","external_server_url":null,"access_token_type":"ALL_ACCOUNTS","name":"ADO","integration_status":null,"access_token_account_name":null,"cloud_integration":true,"created_at":"2026-01-01T00:00:00Z","created_by":"u"}`
	var inst AzureDevopsInstallation
	if err := json.Unmarshal([]byte(fixture), &inst); err != nil {
		t.Fatal(err)
	}
	if inst.AccessTokenType != "ALL_ACCOUNTS" || !inst.CloudIntegration {
		t.Errorf("bad unmarshal: %+v", inst)
	}
}

func TestScmPostureDefaultPolicyWrite_MarshalShape(t *testing.T) {
	disabled := true
	raw, _ := json.Marshal(ScmPostureDefaultPolicyWrite{
		Disabled: false,
		PolicyData: ScmPostureDefaultPolicyData{
			Controls: []ScmPostureControlOverride{
				{ID: "ctrl-1", Disabled: &disabled},
				{ID: "ctrl-2", Priority: "HIGH"},
			},
		},
	})
	var got map[string]any
	_ = json.Unmarshal(raw, &got)
	// disabled and policy_data must always be present on the singleton PUT.
	if _, ok := got["disabled"]; !ok {
		t.Errorf("disabled missing: %s", raw)
	}
	controls := got["policy_data"].(map[string]any)["controls"].([]any)
	first := controls[0].(map[string]any)
	if first["id"] != "ctrl-1" || first["disabled"] != true {
		t.Errorf("bad control 0: %v", first)
	}
	if _, ok := first["priority"]; ok {
		t.Errorf("unset priority must be omitted: %v", first)
	}
	second := controls[1].(map[string]any)
	if second["priority"] != "HIGH" {
		t.Errorf("bad control 1: %v", second)
	}
	if _, ok := second["disabled"]; ok {
		t.Errorf("unset disabled must be omitted: %v", second)
	}
}
