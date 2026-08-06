package shift_left_repository

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func TestGithubRepositoriesToListValue(t *testing.T) {
	list, diags := githubRepositoriesToListValue([]api_client.GithubRepositoryListItem{{
		ScmRepository:      api_client.ScmRepository{ID: "r1", RepositoryName: "org/repo", RepositoryURL: "https://github.com/org/repo"},
		AccountID:          "acct-1",
		GithubRepositoryID: 42,
	}})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if len(list.Elements()) != 1 {
		t.Fatalf("expected one repo, got %#v", list)
	}
}

func TestGitlabRepositoriesToListValue(t *testing.T) {
	list, diags := gitlabRepositoriesToListValue([]api_client.GitlabRepositoryListItem{{
		ScmRepository:   api_client.ScmRepository{ID: "r1", RepositoryName: "group/proj"},
		InstallationID:  "inst-1",
		GitlabGroupID:   7,
		GitlabProjectID: 99,
	}})
	if diags.HasError() || len(list.Elements()) != 1 {
		t.Fatalf("got list=%#v diags=%v", list, diags)
	}
}

func TestBitbucketRepositoriesToListValue(t *testing.T) {
	list, diags := bitbucketRepositoriesToListValue([]api_client.BitbucketRepositoryListItem{{
		ScmRepository:         api_client.ScmRepository{ID: "r1", RepositoryName: "ws/repo", Slug: "repo"},
		InstallationID:        "inst-1",
		AccountID:             "ws",
		BitbucketRepositoryID: "bb-1",
	}})
	if diags.HasError() || len(list.Elements()) != 1 {
		t.Fatalf("got list=%#v diags=%v", list, diags)
	}
}

func TestAzureRepositoriesToListValue(t *testing.T) {
	list, diags := azureRepositoriesToListValue([]api_client.AzureRepositoryListItem{{
		ScmRepository:     api_client.ScmRepository{ID: "r1", RepositoryName: "org/repo"},
		InstallationID:    "inst-1",
		AccountName:       "org",
		AzureRepositoryID: "az-1",
	}})
	if diags.HasError() || len(list.Elements()) != 1 {
		t.Fatalf("got list=%#v diags=%v", list, diags)
	}
}
