package shift_left_gitlab_group_test

import (
	"os"
	"strconv"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
)

func requireGitlabGroupTestEnv(t *testing.T) (installationID, gitlabGroupIDEnv, orcaGroupID string) {
	t.Helper()
	installationID = os.Getenv("ORCA_TEST_GL_INSTALLATION_ID")
	gitlabGroupIDEnv = os.Getenv("ORCA_TEST_GL_GITLAB_GROUP_ID")
	orcaGroupID = os.Getenv("ORCA_TEST_GL_GROUP_ID")
	if installationID == "" || (gitlabGroupIDEnv == "" && orcaGroupID == "") {
		t.Skip("ORCA_TEST_GL_INSTALLATION_ID and ORCA_TEST_GL_GITLAB_GROUP_ID (or ORCA_TEST_GL_GROUP_ID) not set")
	}
	return installationID, gitlabGroupIDEnv, orcaGroupID
}

func fetchGitlabGroupForTest(t *testing.T, client *api_client.APIClient, installationID, gitlabGroupIDEnv, orcaGroupID string) *api_client.GitlabGroup {
	t.Helper()
	var original *api_client.GitlabGroup
	var err error
	if gitlabGroupIDEnv != "" {
		n, perr := strconv.ParseInt(gitlabGroupIDEnv, 10, 64)
		if perr != nil {
			t.Fatalf("ORCA_TEST_GL_GITLAB_GROUP_ID: %v", perr)
		}
		original, err = client.FindGitlabGroupByGitlabID(installationID, n)
	} else {
		original, err = client.GetGitlabGroup(installationID, orcaGroupID)
	}
	if err != nil {
		t.Fatalf("snapshot failed: %s", err)
	}
	if original == nil {
		t.Skipf("gitlab group not found under installation %s", installationID)
	}
	return original
}

// restoreGitlabGroup re-integrates the unit if destroy removed it, then restores config.
func restoreGitlabGroup(t *testing.T, client *api_client.APIClient, installationID string, gitlabGroupID int64, original *api_client.GitlabGroup) {
	t.Helper()
	body := acctest.RestoreScmBody(original.InstallationMode, original.DefaultPolicies, original.Policies, original.Project, original.ConfigSettings)
	client.InvalidateScmListCache()
	cur, err := client.FindGitlabGroupByGitlabID(installationID, gitlabGroupID)
	if err != nil {
		t.Errorf("restore lookup: %s", err)
		return
	}
	if cur == nil {
		if err := client.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
			InstallationID: installationID,
			GitlabGroupID:  gitlabGroupID,
			Body:           body,
		}); err != nil {
			t.Errorf("failed to re-integrate gitlab group %d: %s", gitlabGroupID, err)
		}
		return
	}
	if _, err := client.UpdateGitlabGroup(installationID, cur.ID, body); err != nil {
		client.InvalidateScmListCache()
		if err2 := client.IntegrateGitlabUnit(api_client.GitlabUnitIntegrate{
			InstallationID: installationID,
			GitlabGroupID:  gitlabGroupID,
			Body:           body,
		}); err2 != nil {
			t.Errorf("failed to restore gitlab group %s (update: %v; re-integrate: %v)", cur.ID, err, err2)
		}
	}
}
