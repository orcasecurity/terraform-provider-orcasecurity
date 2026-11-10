package shift_left_repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const movePath = "/api/shiftleft/repository_contexts/move_project/"

// The API refuses a context with a pending move or one already in the target; the async part clears after asyncSleeps waits.
type fakeMoveBackend struct {
	project             string
	pending             bool
	asyncSleeps         int
	sleepsLeft          int
	gatewayTimeouts     int
	commitBeforeTimeout bool
	moves               []string
	finds               int
	sleeps              int
}

func moveErrorBody(code string) string {
	return fmt.Sprintf(`{"status":"failure","error_code":%q,"message":"refused","errors":{}}`, code)
}

func (f *fakeMoveBackend) handle(req *http.Request) *http.Response {
	if req.Method != http.MethodPost || req.URL.Path != movePath {
		return testutils.JSONResponse(req, http.StatusNotFound, `{}`)
	}
	var body struct {
		TargetProjectID string `json:"target_project_id"`
	}
	_ = json.NewDecoder(req.Body).Decode(&body)
	f.moves = append(f.moves, body.TargetProjectID)
	if f.gatewayTimeouts > 0 {
		f.gatewayTimeouts--
		if f.commitBeforeTimeout {
			f.commit(body.TargetProjectID)
		}
		return testutils.JSONResponse(req, http.StatusGatewayTimeout, `upstream timed out`)
	}
	if f.pending {
		return testutils.JSONResponse(req, http.StatusBadRequest, moveErrorBody("move_already_in_progress"))
	}
	if f.project == body.TargetProjectID {
		return testutils.JSONResponse(req, http.StatusBadRequest,
			moveErrorBody("repositories_contexts_already_exists_in_target_project"))
	}
	f.commit(body.TargetProjectID)
	return testutils.JSONResponse(req, http.StatusOK, ``)
}

func (f *fakeMoveBackend) commit(target string) {
	f.project = target
	f.pending = f.asyncSleeps > 0
	f.sleepsLeft = f.asyncSleeps
}

func (f *fakeMoveBackend) ops(t *testing.T) repoOps {
	t.Helper()
	origDelays, origSleep := moveRetryDelays, moveSleep
	moveRetryDelays = []time.Duration{0, time.Millisecond, time.Millisecond, time.Millisecond, time.Millisecond}
	moveSleep = func(ctx context.Context, d time.Duration) error {
		if d > 0 {
			f.sleeps++
			if f.pending {
				f.sleepsLeft--
				f.pending = f.sleepsLeft > 0
			}
		}
		return ctx.Err()
	}
	t.Cleanup(func() { moveRetryDelays, moveSleep = origDelays, origSleep })
	return repoOps{
		client: testutils.NewStubAPIClient(f.handle),
		traits: githubTraits,
		find: func() (*api_client.ScmRepository, error) {
			f.finds++
			return &api_client.ScmRepository{ID: "row-1", ProjectID: f.project, RepositoryContextID: "ctx-1"}, nil
		},
		update: func(api_client.ScmRepositoryConfigUpdate) error { return nil },
	}
}

func TestMoveRepository_ReplayRefusedAlreadyInTargetIsConfirmedByReRead(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-new"}
	if err := moveRepository(context.Background(), f.ops(t), "ctx-1", "proj-new"); err != nil {
		t.Fatalf("a repository already in the target project is a completed move, got %v", err)
	}
	if len(f.moves) != 1 || f.finds != 1 {
		t.Fatalf("expected one move and one confirming read, moves=%v finds=%d", f.moves, f.finds)
	}
}

func TestMoveRepository_AlreadyInTargetRefusalNotTrustedWithoutReRead(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old"}
	o := f.ops(t)
	o.client = testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		f.moves = append(f.moves, "proj-new")
		return testutils.JSONResponse(req, http.StatusBadRequest,
			moveErrorBody("repositories_contexts_already_exists_in_target_project"))
	})
	err := moveRepository(context.Background(), o, "ctx-1", "proj-new")
	if !errors.Is(err, api_client.ErrMoveAlreadyInTarget) || !strings.Contains(err.Error(), "proj-old") {
		t.Fatalf("expected the refusal reported with the project the read shows, got %v", err)
	}
}

func TestMoveRepository_InProgressOfOwnCommittedMoveIsSuccess(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-new", pending: true, asyncSleeps: 3, sleepsLeft: 3}
	if err := moveRepository(context.Background(), f.ops(t), "ctx-1", "proj-new"); err != nil {
		t.Fatalf("a pending move into the target project is a completed move, got %v", err)
	}
	if len(f.moves) != 1 || f.sleeps != 0 {
		t.Fatalf("no wait expected once the read shows the target project, moves=%v sleeps=%d", f.moves, f.sleeps)
	}
}

func TestMoveRepository_WaitsForEarlierMoveThenMoves(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-other", pending: true, sleepsLeft: 2}
	if err := moveRepository(context.Background(), f.ops(t), "ctx-1", "proj-new"); err != nil {
		t.Fatalf("expected the move to go through once the earlier move finished, got %v", err)
	}
	if f.project != "proj-new" || len(f.moves) != 3 {
		t.Fatalf("expected two refused attempts then the move, project=%s moves=%v", f.project, f.moves)
	}
}

func TestMoveRepository_AmbiguousCommittedIsConfirmedByReRead(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old", asyncSleeps: 5, gatewayTimeouts: 1, commitBeforeTimeout: true}
	if err := moveRepository(context.Background(), f.ops(t), "ctx-1", "proj-new"); err != nil {
		t.Fatalf("a 504 after the move committed must be confirmed by the read, got %v", err)
	}
	if len(f.moves) != 1 || f.finds != 1 {
		t.Fatalf("the move must not be resent once the read shows it landed, moves=%v finds=%d", f.moves, f.finds)
	}
}

func TestMoveRepository_AmbiguousNotCommittedIsResent(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old", gatewayTimeouts: 1}
	if err := moveRepository(context.Background(), f.ops(t), "ctx-1", "proj-new"); err != nil {
		t.Fatalf("expected the resent move to land, got %v", err)
	}
	if f.project != "proj-new" || len(f.moves) != 2 || f.sleeps != 1 {
		t.Fatalf("expected one resend after a wait, project=%s moves=%v sleeps=%d", f.project, f.moves, f.sleeps)
	}
}

func TestMoveRepository_OtherRefusalFailsWithoutReRead(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old"}
	o := f.ops(t)
	o.client = testutils.NewStubAPIClient(func(req *http.Request) *http.Response {
		return testutils.JSONResponse(req, http.StatusBadRequest, moveErrorBody("move_repository_context_error"))
	})
	if err := moveRepository(context.Background(), o, "ctx-1", "proj-missing"); err == nil {
		t.Fatal("expected a definite refusal to fail the move")
	}
	if f.finds != 0 {
		t.Fatalf("a definite refusal needs no read, finds=%d", f.finds)
	}
}

func TestMoveRepository_StopsWaitingWhenContextCancelled(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-other", pending: true, sleepsLeft: 100}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := f.ops(t)
	inner := moveSleep
	moveSleep = func(c context.Context, d time.Duration) error {
		if d > 0 {
			cancel()
		}
		return inner(c, d)
	}
	err := moveRepository(ctx, o, "ctx-1", "proj-new")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, api_client.ErrMoveInProgress) {
		t.Fatalf("expected the in-progress refusal plus the cancellation, got %v", err)
	}
	if len(f.moves) != 1 {
		t.Fatalf("no further attempts after cancellation, moves=%v", f.moves)
	}
}

func failingConfigUpdate(t *testing.T, f *fakeMoveBackend) (*diag.Diagnostics, *api_client.ScmRepository) {
	t.Helper()
	o := f.ops(t)
	o.update = func(api_client.ScmRepositoryConfigUpdate) error { return errors.New("config rejected") }
	plan := &RepoConfigFields{ProjectID: types.StringValue("proj-new"), Disabled: types.BoolValue(true)}
	state := &RepoConfigFields{
		ID:                  types.StringValue("row-1"),
		ProjectID:           types.StringValue("proj-old"),
		RepositoryContextID: types.StringValue("ctx-1"),
	}
	var diags diag.Diagnostics
	row := updateRepo(context.Background(), o, plan, state, &diags)
	return &diags, row
}

func TestUpdateRepo_RollbackWaitsForInProgressMoveThenMovesBack(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old", asyncSleeps: 2}
	diags, row := failingConfigUpdate(t, f)
	if row != nil || !diags.HasError() {
		t.Fatalf("the config failure must still fail the update, row=%+v diags=%v", row, *diags)
	}
	if diags.WarningsCount() != 0 {
		t.Fatalf("a completed rollback needs no warning, got %v", diags.Warnings())
	}
	if f.project != "proj-old" {
		t.Fatalf("expected the repository back in proj-old, it is in %s (moves=%v)", f.project, f.moves)
	}
	want := []string{"proj-new", "proj-old", "proj-old", "proj-old"}
	if strings.Join(f.moves, ",") != strings.Join(want, ",") {
		t.Fatalf("moves = %v, want %v", f.moves, want)
	}
}

func TestUpdateRepo_RollbackGivesUpWithPlacementWarning(t *testing.T) {
	f := &fakeMoveBackend{project: "proj-old", asyncSleeps: 100}
	diags, _ := failingConfigUpdate(t, f)
	if !diags.HasError() || diags.WarningsCount() != 1 {
		t.Fatalf("expected the config error plus one rollback warning, got %v", *diags)
	}
	w := diags.Warnings()[0]
	if !strings.Contains(w.Summary(), "left in project proj-new") {
		t.Errorf("summary must name where the repository is: %q", w.Summary())
	}
	for _, s := range []string{"moving it back to project proj-old", "stays in project proj-new", "Orca console"} {
		if !strings.Contains(w.Detail(), s) {
			t.Errorf("detail missing %q: %q", s, w.Detail())
		}
	}
	if f.project != "proj-new" {
		t.Fatalf("the repository should be left in proj-new, it is in %s", f.project)
	}
}
