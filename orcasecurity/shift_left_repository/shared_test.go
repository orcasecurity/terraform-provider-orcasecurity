package shift_left_repository

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKnown(t *testing.T) {
	cases := []struct {
		name string
		val  interface {
			IsNull() bool
			IsUnknown() bool
		}
		want bool
	}{
		{"null string", types.StringNull(), false},
		{"unknown string", types.StringUnknown(), false},
		{"known string", types.StringValue("x"), true},
		{"known empty string", types.StringValue(""), true},
		{"null bool", types.BoolNull(), false},
		{"unknown bool", types.BoolUnknown(), false},
		{"known bool", types.BoolValue(false), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := known(tc.val); got != tc.want {
				t.Errorf("known(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestFromAPI_MapsAllFields(t *testing.T) {
	dspr := true
	api := &api_client.ScmRepository{
		ID:                  "row-1",
		RepositoryName:      "acme/repo",
		RepositoryURL:       "https://example.com/acme/repo",
		ProjectID:           "proj-1",
		Disabled:            true,
		DisableScanPRs:      &dspr,
		CommentsOnPRs:       "NEVER",
		PrSummaryComment:    "ALWAYS",
		SkipCheckRuns:       "NEVER",
		ConfigFileSupport:   "ENABLED",
		Status:              "SUCCESS",
		RepositoryContextID: "ctx-1",
		IntegrationStatus:   "INSTALLATION_UNREACHABLE",
		ScmPosturePolicyID:  "sp-1",
	}
	prior := RepoConfigFields{Branch: types.StringValue("main")}
	out := fromAPI(prior, api)

	if out.ID.ValueString() != "row-1" || out.Name.ValueString() != "acme/repo" || out.URL.ValueString() != "https://example.com/acme/repo" {
		t.Errorf("identity mapping wrong: %+v", out)
	}
	// Branch is create-only; it is preserved from prior state, never from the API.
	if out.Branch.ValueString() != "main" {
		t.Errorf("branch must be preserved from prior, got %v", out.Branch)
	}
	if out.ProjectID.ValueString() != "proj-1" || !out.Disabled.ValueBool() {
		t.Errorf("project/disabled wrong: %+v", out)
	}
	if !out.DisableScanPullRequests.ValueBool() {
		t.Errorf("disable_scan_pull_requests should be true, got %v", out.DisableScanPullRequests)
	}
	if out.CommentsOnPullRequests.ValueString() != "NEVER" || out.PrSummaryComment.ValueString() != "ALWAYS" {
		t.Errorf("pr comment fields wrong: %+v", out)
	}
	if out.SkipCheckRuns.ValueString() != "NEVER" || out.ConfigFileSupport.ValueString() != "ENABLED" {
		t.Errorf("skip/config fields wrong: %+v", out)
	}
	if out.Status.ValueString() != "SUCCESS" || out.RepositoryContextID.ValueString() != "ctx-1" {
		t.Errorf("status/ctx wrong: %+v", out)
	}
	if out.IntegrationStatus.ValueString() != "INSTALLATION_UNREACHABLE" || out.ScmPosturePolicyID.ValueString() != "sp-1" {
		t.Errorf("integration/posture wrong: %+v", out)
	}
}

func TestFromAPI_EmptyStringsBecomeNull(t *testing.T) {
	// OptionalID maps "" to null so unset optional/computed strings do not flap.
	out := fromAPI(RepoConfigFields{}, &api_client.ScmRepository{ID: "row-1", RepositoryName: "n", RepositoryURL: "u"})
	nulls := map[string]types.String{
		"project_id":                out.ProjectID,
		"comments_on_pull_requests": out.CommentsOnPullRequests,
		"pr_summary_comment":        out.PrSummaryComment,
		"skip_check_runs":           out.SkipCheckRuns,
		"config_file_support":       out.ConfigFileSupport,
		"status":                    out.Status,
		"repository_context_id":     out.RepositoryContextID,
		"integration_status":        out.IntegrationStatus,
		"scm_posture_policy_id":     out.ScmPosturePolicyID,
	}
	for name, v := range nulls {
		if !v.IsNull() {
			t.Errorf("%s should be null when API returns empty, got %#v", name, v)
		}
	}
}

func TestFromAPI_DisableScanPRsNilHandling(t *testing.T) {
	cases := []struct {
		name     string
		api      *bool
		wantNull bool
		wantVal  bool
	}{
		{"nil -> null", nil, true, false},
		{"true -> true", boolPtr(true), false, true},
		{"false -> false", boolPtr(false), false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := fromAPI(RepoConfigFields{}, &api_client.ScmRepository{DisableScanPRs: tc.api})
			if out.DisableScanPullRequests.IsNull() != tc.wantNull {
				t.Fatalf("null=%v, want %v", out.DisableScanPullRequests.IsNull(), tc.wantNull)
			}
			if !tc.wantNull && out.DisableScanPullRequests.ValueBool() != tc.wantVal {
				t.Fatalf("value=%v, want %v", out.DisableScanPullRequests.ValueBool(), tc.wantVal)
			}
		})
	}
}

func TestFromAPI_SkipCheckRunsAzureFallback(t *testing.T) {
	cases := []struct {
		name     string
		apiValue string
		prior    types.String
		want     types.String
	}{
		// Azure's list serializer omits skip_check_runs; keep the last written value.
		{"empty api + known prior keeps prior", "", types.StringValue("ALWAYS"), types.StringValue("ALWAYS")},
		{"empty api + null prior stays null", "", types.StringNull(), types.StringNull()},
		{"empty api + unknown prior stays null", "", types.StringUnknown(), types.StringNull()},
		{"non-empty api overrides prior", "NEVER", types.StringValue("ALWAYS"), types.StringValue("NEVER")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := fromAPI(RepoConfigFields{SkipCheckRuns: tc.prior}, &api_client.ScmRepository{SkipCheckRuns: tc.apiValue})
			if tc.want.IsNull() {
				if !out.SkipCheckRuns.IsNull() {
					t.Fatalf("expected null skip_check_runs, got %#v", out.SkipCheckRuns)
				}
				return
			}
			if out.SkipCheckRuns.ValueString() != tc.want.ValueString() {
				t.Fatalf("skip_check_runs = %v, want %v", out.SkipCheckRuns, tc.want)
			}
		})
	}
}

func TestConfigUpdateBody(t *testing.T) {
	t.Run("nothing set", func(t *testing.T) {
		body, set := configUpdateBody("row-1", &RepoConfigFields{})
		if set {
			t.Errorf("no known fields must yield set=false, got body %+v", body)
		}
		if len(body.IDs) != 1 || body.IDs[0] != "row-1" {
			t.Errorf("IDs must always carry the row id: %+v", body.IDs)
		}
	})

	t.Run("all fields set", func(t *testing.T) {
		plan := &RepoConfigFields{
			Disabled:                types.BoolValue(true),
			DisableScanPullRequests: types.BoolValue(false),
			CommentsOnPullRequests:  types.StringValue("ALWAYS"),
			PrSummaryComment:        types.StringValue("NEVER"),
			SkipCheckRuns:           types.StringValue("ALWAYS"),
			ConfigFileSupport:       types.StringValue("ENABLED"),
		}
		body, set := configUpdateBody("row-1", plan)
		if !set {
			t.Fatal("expected set=true")
		}
		if body.Disabled == nil || *body.Disabled != true {
			t.Errorf("disabled: %v", body.Disabled)
		}
		if body.DisableScanPullRequests == nil || *body.DisableScanPullRequests != false {
			t.Errorf("disable_scan_pull_requests: %v", body.DisableScanPullRequests)
		}
		if body.CommentsOnPullRequests != "ALWAYS" || body.PrSummaryComment != "NEVER" {
			t.Errorf("comment fields: %+v", body)
		}
		if body.SkipCheckRuns != "ALWAYS" || body.ConfigFileSupport != "ENABLED" {
			t.Errorf("skip/config: %+v", body)
		}
	})

	t.Run("null and unknown fields skipped", func(t *testing.T) {
		plan := &RepoConfigFields{
			Disabled:               types.BoolValue(true), // only this is known
			CommentsOnPullRequests: types.StringNull(),
			PrSummaryComment:       types.StringUnknown(),
		}
		body, set := configUpdateBody("row-1", plan)
		if !set {
			t.Fatal("expected set=true (disabled is known)")
		}
		if body.DisableScanPullRequests != nil {
			t.Errorf("unset disable_scan_pull_requests must stay nil: %v", body.DisableScanPullRequests)
		}
		if body.CommentsOnPullRequests != "" || body.PrSummaryComment != "" {
			t.Errorf("null/unknown strings must stay empty: %+v", body)
		}
	})
}

func TestIntegrateConfig(t *testing.T) {
	t.Run("empty plan yields empty config", func(t *testing.T) {
		cfg := integrateConfig(&RepoConfigFields{})
		if cfg.DisableScanPullRequests != nil || cfg.CommentsOnPullRequests != "" ||
			cfg.PrSummaryComment != "" || cfg.SkipCheckRuns != "" || cfg.ConfigFileSupport != "" {
			t.Errorf("expected zero-value config, got %+v", cfg)
		}
	})

	t.Run("known fields propagate, disabled is excluded", func(t *testing.T) {
		// Disabled is intentionally not part of ScmRepoIntegrationConfig (GitHub's
		// integrate endpoint rejects it); it is applied post-integrate instead.
		plan := &RepoConfigFields{
			Disabled:                types.BoolValue(true),
			DisableScanPullRequests: types.BoolValue(true),
			CommentsOnPullRequests:  types.StringValue("NEVER"),
			PrSummaryComment:        types.StringValue("ALWAYS"),
			SkipCheckRuns:           types.StringValue("NEVER"),
			ConfigFileSupport:       types.StringValue("DISABLED"),
		}
		cfg := integrateConfig(plan)
		if cfg.DisableScanPullRequests == nil || !*cfg.DisableScanPullRequests {
			t.Errorf("disable_scan_pull_requests: %v", cfg.DisableScanPullRequests)
		}
		if cfg.CommentsOnPullRequests != "NEVER" || cfg.PrSummaryComment != "ALWAYS" {
			t.Errorf("comment fields: %+v", cfg)
		}
		if cfg.SkipCheckRuns != "NEVER" || cfg.ConfigFileSupport != "DISABLED" {
			t.Errorf("skip/config: %+v", cfg)
		}
	})
}

func TestBranchAttribute(t *testing.T) {
	req := branchAttribute(true)
	if !req.Required || req.Optional {
		t.Errorf("branchRequired=true must be Required, got Required=%v Optional=%v", req.Required, req.Optional)
	}
	opt := branchAttribute(false)
	if opt.Required || !opt.Optional {
		t.Errorf("branchRequired=false must be Optional, got Required=%v Optional=%v", opt.Required, opt.Optional)
	}
	// Both variants force replacement (create-only field).
	if len(req.PlanModifiers) == 0 || len(opt.PlanModifiers) == 0 {
		t.Error("branch must carry a RequiresReplace plan modifier in both variants")
	}
}

func TestSharedRepoAttributes(t *testing.T) {
	attrs := sharedRepoAttributes("GitHub", fullSkipCheckRuns, true)
	wantKeys := []string{
		"id", "name", "url", "branch", "project_id", "disabled",
		"disable_scan_pull_requests", "comments_on_pull_requests", "pr_summary_comment",
		"skip_check_runs", "config_file_support", "status", "repository_context_id",
		"integration_status", "scm_posture_policy_id",
	}
	for _, k := range wantKeys {
		if _, ok := attrs[k]; !ok {
			t.Errorf("missing attribute %q", k)
		}
	}
	if attrs["project_id"].IsRequired() {
		t.Error("project_id must be optional (Optional+Computed)")
	}
	if !attrs["project_id"].IsOptional() || !attrs["project_id"].IsComputed() {
		t.Error("project_id must be Optional and Computed")
	}
	if !attrs["name"].IsRequired() {
		t.Error("name must be required")
	}
	if !attrs["id"].IsComputed() || attrs["id"].IsRequired() {
		t.Error("id must be computed-only")
	}
}

// --- createRepo / updateRepo / deleteRepo decision logic ---

func boolPtr(b bool) *bool { return &b }

// recordingClient returns an APIClient whose requests are all answered with the
// given status and an empty JSON body, plus a pointer to the recorded
// "METHOD path" strings.
func recordingClient(status int) (*api_client.APIClient, *[]string) {
	var reqs []string
	c := testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		reqs = append(reqs, req.Method+" "+req.URL.Path)
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader("{}")),
			Header:     make(http.Header),
		}
	})
	return c, &reqs
}

func TestCreateRepo_IntegrateError(t *testing.T) {
	ops := repoOps{
		scmName:   "GitHub",
		integrate: func() error { return errors.New("boom") },
		find:      func() (*api_client.ScmRepository, error) { t.Fatal("find must not run"); return nil, nil },
	}
	var diags diag.Diagnostics
	if row := createRepo(ops, &RepoConfigFields{}, &diags); row != nil {
		t.Fatalf("expected nil row on integrate error, got %+v", row)
	}
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic")
	}
}

func TestCreateRepo_NotFoundAfterIntegrate(t *testing.T) {
	ops := repoOps{
		scmName:   "GitHub",
		integrate: func() error { return nil },
		find:      func() (*api_client.ScmRepository, error) { return nil, nil },
	}
	var diags diag.Diagnostics
	if row := createRepo(ops, &RepoConfigFields{}, &diags); row != nil {
		t.Fatalf("expected nil row when not found, got %+v", row)
	}
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for missing repo")
	}
}

func TestCreateRepo_HappyNoConfig(t *testing.T) {
	found := &api_client.ScmRepository{ID: "row-1", RepositoryContextID: "ctx-1"}
	findCalls := 0
	ops := repoOps{
		scmName:   "GitHub",
		integrate: func() error { return nil },
		find: func() (*api_client.ScmRepository, error) {
			findCalls++
			return found, nil
		},
		update: func(api_client.ScmRepositoryConfigUpdate) error { t.Fatal("update must not run"); return nil },
	}
	var diags diag.Diagnostics
	row := createRepo(ops, &RepoConfigFields{}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if row != found || findCalls != 1 {
		t.Fatalf("expected single find and returned row, findCalls=%d row=%+v", findCalls, row)
	}
}

func TestCreateRepo_ConfigAppliedRereads(t *testing.T) {
	first := &api_client.ScmRepository{ID: "row-1", Status: "IN_PROGRESS"}
	second := &api_client.ScmRepository{ID: "row-1", Status: "SUCCESS"}
	findCalls := 0
	updateCalls := 0
	ops := repoOps{
		scmName:   "GitHub",
		integrate: func() error { return nil },
		find: func() (*api_client.ScmRepository, error) {
			findCalls++
			if findCalls == 1 {
				return first, nil
			}
			return second, nil
		},
		update: func(b api_client.ScmRepositoryConfigUpdate) error {
			updateCalls++
			if len(b.IDs) != 1 || b.IDs[0] != "row-1" {
				t.Errorf("update body must target the row: %+v", b.IDs)
			}
			return nil
		},
	}
	var diags diag.Diagnostics
	row := createRepo(ops, &RepoConfigFields{Disabled: types.BoolValue(true)}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if updateCalls != 1 || findCalls != 2 || row != second {
		t.Fatalf("expected update + re-read, updateCalls=%d findCalls=%d row=%+v", updateCalls, findCalls, row)
	}
}

func TestCreateRepo_ConfigFailureRollsBack(t *testing.T) {
	client, reqs := recordingClient(http.StatusOK)
	ops := repoOps{
		client:    client,
		scmName:   "GitHub",
		integrate: func() error { return nil },
		find: func() (*api_client.ScmRepository, error) {
			return &api_client.ScmRepository{ID: "row-1", RepositoryContextID: "ctx-1"}, nil
		},
		update: func(api_client.ScmRepositoryConfigUpdate) error { return errors.New("config rejected") },
	}
	var diags diag.Diagnostics
	row := createRepo(ops, &RepoConfigFields{Disabled: types.BoolValue(true)}, &diags)
	if row != nil {
		t.Fatalf("expected nil row after config failure, got %+v", row)
	}
	if !diags.HasError() {
		t.Fatal("expected a create error diagnostic")
	}
	if len(*reqs) != 1 || (*reqs)[0] != "DELETE /api/shiftleft/repository_contexts/ctx-1/" {
		t.Fatalf("rollback must DELETE the repository context, recorded: %v", *reqs)
	}
}

func TestCreateRepo_ConfigFailureNoContextSkipsRollback(t *testing.T) {
	// No RepositoryContextID means rollback has nothing to delete; it must not
	// dereference the (nil) client.
	ops := repoOps{
		scmName:   "GitHub",
		integrate: func() error { return nil },
		find:      func() (*api_client.ScmRepository, error) { return &api_client.ScmRepository{ID: "row-1"}, nil },
		update:    func(api_client.ScmRepositoryConfigUpdate) error { return errors.New("config rejected") },
	}
	var diags diag.Diagnostics
	row := createRepo(ops, &RepoConfigFields{Disabled: types.BoolValue(true)}, &diags)
	if row != nil || !diags.HasError() {
		t.Fatalf("expected nil row + error, got row=%+v hasErr=%v", row, diags.HasError())
	}
}

func TestUpdateRepo_NoConfigNoMove(t *testing.T) {
	found := &api_client.ScmRepository{ID: "row-1"}
	ops := repoOps{
		scmName: "GitHub",
		find:    func() (*api_client.ScmRepository, error) { return found, nil },
		update:  func(api_client.ScmRepositoryConfigUpdate) error { t.Fatal("update must not run"); return nil },
	}
	plan := &RepoConfigFields{}
	state := &RepoConfigFields{ID: types.StringValue("row-1")}
	var diags diag.Diagnostics
	if row := updateRepo(ops, plan, state, &diags); row != found || diags.HasError() {
		t.Fatalf("expected clean re-read, row=%+v hasErr=%v", row, diags.HasError())
	}
}

func TestUpdateRepo_ConfigUpdateError(t *testing.T) {
	ops := repoOps{
		scmName: "GitHub",
		update:  func(api_client.ScmRepositoryConfigUpdate) error { return errors.New("nope") },
		find: func() (*api_client.ScmRepository, error) {
			t.Fatal("find must not run after update error")
			return nil, nil
		},
	}
	plan := &RepoConfigFields{Disabled: types.BoolValue(true)}
	state := &RepoConfigFields{ID: types.StringValue("row-1")}
	var diags diag.Diagnostics
	if row := updateRepo(ops, plan, state, &diags); row != nil || !diags.HasError() {
		t.Fatalf("expected nil row + error, got row=%+v hasErr=%v", row, diags.HasError())
	}
}

func TestUpdateRepo_MoveWithoutContextIDErrors(t *testing.T) {
	ops := repoOps{scmName: "GitHub"}
	plan := &RepoConfigFields{ProjectID: types.StringValue("proj-new")}
	state := &RepoConfigFields{ID: types.StringValue("row-1"), ProjectID: types.StringValue("proj-old")}
	var diags diag.Diagnostics
	if row := updateRepo(ops, plan, state, &diags); row != nil || !diags.HasError() {
		t.Fatalf("expected error when context id is unknown, got row=%+v hasErr=%v", row, diags.HasError())
	}
}

func TestUpdateRepo_MovesProject(t *testing.T) {
	client, reqs := recordingClient(http.StatusOK)
	found := &api_client.ScmRepository{ID: "row-1", ProjectID: "proj-new"}
	ops := repoOps{
		client:  client,
		scmName: "GitHub",
		find:    func() (*api_client.ScmRepository, error) { return found, nil },
	}
	plan := &RepoConfigFields{ProjectID: types.StringValue("proj-new")}
	state := &RepoConfigFields{
		ID:                  types.StringValue("row-1"),
		ProjectID:           types.StringValue("proj-old"),
		RepositoryContextID: types.StringValue("ctx-1"),
	}
	var diags diag.Diagnostics
	row := updateRepo(ops, plan, state, &diags)
	if diags.HasError() || row != found {
		t.Fatalf("expected clean move, row=%+v hasErr=%v", row, diags.HasError())
	}
	if len(*reqs) != 1 || (*reqs)[0] != "POST /api/shiftleft/repository_contexts/move_project/" {
		t.Fatalf("expected a move_project POST, recorded: %v", *reqs)
	}
}

func TestUpdateRepo_SameProjectDoesNotMove(t *testing.T) {
	client, reqs := recordingClient(http.StatusOK)
	found := &api_client.ScmRepository{ID: "row-1", ProjectID: "proj-1"}
	ops := repoOps{
		client:  client,
		scmName: "GitHub",
		find:    func() (*api_client.ScmRepository, error) { return found, nil },
	}
	plan := &RepoConfigFields{ProjectID: types.StringValue("proj-1")}
	state := &RepoConfigFields{ID: types.StringValue("row-1"), ProjectID: types.StringValue("proj-1")}
	var diags diag.Diagnostics
	if row := updateRepo(ops, plan, state, &diags); row != found || diags.HasError() {
		t.Fatalf("unexpected: row=%+v hasErr=%v", row, diags.HasError())
	}
	if len(*reqs) != 0 {
		t.Fatalf("no move expected when project unchanged, recorded: %v", *reqs)
	}
}

func TestDeleteRepo_UsesStateContextID(t *testing.T) {
	client, reqs := recordingClient(http.StatusOK)
	ops := repoOps{
		client:  client,
		scmName: "GitHub",
		find: func() (*api_client.ScmRepository, error) {
			t.Fatal("find must not run when ctx id is in state")
			return nil, nil
		},
	}
	state := &RepoConfigFields{RepositoryContextID: types.StringValue("ctx-1")}
	var diags diag.Diagnostics
	deleteRepo(ops, state, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(*reqs) != 1 || (*reqs)[0] != "DELETE /api/shiftleft/repository_contexts/ctx-1/" {
		t.Fatalf("expected DELETE of ctx-1, recorded: %v", *reqs)
	}
}

func TestDeleteRepo_FallsBackToFind(t *testing.T) {
	client, reqs := recordingClient(http.StatusOK)
	ops := repoOps{
		client:  client,
		scmName: "GitHub",
		find: func() (*api_client.ScmRepository, error) {
			return &api_client.ScmRepository{ID: "row-1", RepositoryContextID: "ctx-live"}, nil
		},
	}
	// Empty state ctx id (state predates the field): fall back to a live read.
	state := &RepoConfigFields{}
	var diags diag.Diagnostics
	deleteRepo(ops, state, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(*reqs) != 1 || (*reqs)[0] != "DELETE /api/shiftleft/repository_contexts/ctx-live/" {
		t.Fatalf("expected DELETE of ctx-live from fallback read, recorded: %v", *reqs)
	}
}

func TestDeleteRepo_AlreadyGone(t *testing.T) {
	// Empty state ctx id + find returns nil: nothing to delete, no client call.
	ops := repoOps{
		scmName: "GitHub",
		find:    func() (*api_client.ScmRepository, error) { return nil, nil },
	}
	var diags diag.Diagnostics
	deleteRepo(ops, &RepoConfigFields{}, &diags)
	if diags.HasError() {
		t.Fatalf("already-gone delete must be a no-op, got %v", diags)
	}
}

func TestDeleteRepo_FindErrorSurfaces(t *testing.T) {
	ops := repoOps{
		scmName: "GitHub",
		find:    func() (*api_client.ScmRepository, error) { return nil, errors.New("read failed") },
	}
	var diags diag.Diagnostics
	deleteRepo(ops, &RepoConfigFields{}, &diags)
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic when the fallback read fails")
	}
}
