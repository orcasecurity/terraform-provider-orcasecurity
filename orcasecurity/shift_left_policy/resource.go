package shift_left_policy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &shiftLeftPolicyResource{}
	_ resource.ResourceWithConfigure      = &shiftLeftPolicyResource{}
	_ resource.ResourceWithImportState    = &shiftLeftPolicyResource{}
	_ resource.ResourceWithValidateConfig = &shiftLeftPolicyResource{}
	_ resource.ResourceWithModifyPlan     = &shiftLeftPolicyResource{}
)

type shiftLeftPolicyResource struct {
	apiClient *api_client.APIClient
}

func NewShiftLeftPolicyResource() resource.Resource {
	return &shiftLeftPolicyResource{}
}

func (r *shiftLeftPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_policy"
}

func (r *shiftLeftPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *shiftLeftPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an AppSec (Shift Left) policy resource. Use this resource to create and manage AppSec scan policies in Orca Security.",
		Attributes:  resourceSchemaAttributes(),
		Blocks:      resourceSchemaBlocks(),
	}
}

// ValidateConfig rejects combinations the API cannot apply. scm_posture policies are scoped via
// scm_posture.scope; the API has no projects endpoint for them and its serializer silently drops
// projects_ids. Older provider versions sent it in the body and applied cleanly, so projects_ids is
// a warning (and a no-op) here rather than an error, to keep those configs planning.
func (r *shiftLeftPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	attachAll := tfconv.BoolIsTrue(config.AttachAllProjects)
	if attachAll && !config.ProjectsIds.IsNull() && !config.ProjectsIds.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("attach_all_projects"),
			"Conflicting project attachment",
			"attach_all_projects and projects_ids cannot both be set — the API rejects a request carrying both. "+
				"Use attach_all_projects to attach every project in the organization, or projects_ids to attach a specific set.",
		)
	}
	if config.Type.IsUnknown() {
		return
	}
	unsupported := projectsUnsupported(config.Type.ValueString())
	if unsupported && attachAll {
		resp.Diagnostics.AddAttributeError(
			path.Root("attach_all_projects"),
			"Unsupported attribute for scm_posture",
			"scm_posture policies cannot use attach_all_projects — the API has no project-attachment endpoint for this type. "+
				"Scope them with scm_posture.scope (installation/unit IDs) instead.",
		)
	}
	if config.ProjectsIds.IsUnknown() {
		return
	}
	if unsupported && !config.ProjectsIds.IsNull() {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("projects_ids"),
			"projects_ids is ignored for scm_posture",
			"scm_posture policies cannot be attached to projects — the API ignores projects_ids for this type, so it has no effect. "+
				"Scope them with scm_posture.scope (installation/unit IDs) instead, and remove projects_ids; it will become an error in a future release.",
		)
	}
}

// ModifyPlan keeps attach_all_projects honest across applies. The API resolves the project set at
// request time, so a project created in Orca after the last apply is not attached yet even though
// nothing in the configuration changed. Comparing the attached set in state against the live project
// list turns that into a planned change; leaving projects_ids unknown (rather than planning the
// enumerated list) means a project created between plan and apply widens the set without tripping
// Terraform's "provider produced inconsistent result" check.
func (r *shiftLeftPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return // destroy, or create (projects_ids is already unknown)
	}

	var plan shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !tfconv.BoolIsTrue(plan.AttachAllProjects) {
		return
	}

	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projects, err := r.apiClient.ListShiftLeftProjects()
	if err != nil {
		// A plan must not fail because the project list is momentarily unavailable. Re-attaching is
		// idempotent, so the safe fallback is to plan the attach and let apply resolve the real set.
		resp.Diagnostics.AddWarning(
			"Could not check for newly added projects",
			fmt.Sprintf("Listing organization projects to plan attach_all_projects failed: %s. "+
				"Planning the attachment anyway; it is a no-op when every project is already attached.", err.Error()),
		)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("projects_ids"), types.ListUnknown(types.StringType))...)
		return
	}

	if attachedEveryProject(tfconv.ListToStringSlice(state.ProjectsIds), projects) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("projects_ids"), types.ListUnknown(types.StringType))...)
}

// attachedEveryProject reports whether attached already covers every project the org has. It ignores
// extras on either side of the comparison's direction that don't matter: a project deleted in Orca
// still lingering in state is not a reason to re-attach.
func attachedEveryProject(attached []string, projects []api_client.ShiftLeftProjectSummary) bool {
	have := make(map[string]bool, len(attached))
	for _, id := range attached {
		have[id] = true
	}
	for _, project := range projects {
		if !have[project.ID] {
			return false
		}
	}
	return true
}

func parseImportID(id string) (policyType, policyID string, err error) {
	policyType, policyID, ok := strings.Cut(id, "/")
	if !ok || policyType == "" || policyID == "" {
		return "", "", fmt.Errorf("import ID must be in the format <type>/<id>, got %q", id)
	}
	return policyType, policyID, nil
}

func (r *shiftLeftPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyType, policyID, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing AppSec policy", err.Error())
		return
	}
	if instance == nil {
		resp.Diagnostics.AddError("Error importing AppSec policy", fmt.Sprintf("Policy %s/%s not found.", policyType, policyID))
		return
	}
	// The org-wide built-in scm_posture policy has a dedicated singleton resource with
	// its own PUT endpoint and locked fields. Owning it from two resource types at once
	// invites conflicting writes, so this resource refuses to take it.
	if policyType == "scm_posture" && instance.Builtin {
		resp.Diagnostics.AddError(
			"Built-in scm_posture policy has a dedicated resource",
			"The org-wide built-in SCM posture policy is managed exclusively by "+
				"orcasecurity_shift_left_scm_posture_default_policy, which adopts it on apply without "+
				"an import. Use orcasecurity_shift_left_policy only for custom scm_posture policies.",
		)
		return
	}

	state := apiToState(instance, nil)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), state.ID.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), policyType)...)
}

func (r *shiftLeftPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiPolicy, diags := planToAPI(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyType := plan.Type.ValueString()
	// Projects sync via dedicated endpoint — main body omitempty drops empty projects_ids.
	apiPolicy.ProjectsIds = nil
	if !r.applyCatalog(&plan, &apiPolicy, &resp.Diagnostics) {
		return
	}

	preexisting, snapshotErr := r.apiClient.ListShiftLeftPolicyIDsByName(policyType, apiPolicy.Name)
	if snapshotErr != nil {
		tflog.Warn(ctx, "could not snapshot same-name policies before create; an ambiguous create will not be auto-recovered: "+snapshotErr.Error())
		preexisting = nil
	}
	policyID, err := r.createPolicy(ctx, policyType, apiPolicy, preexisting, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AppSec policy", "Could not create policy: "+err.Error())
		return
	}

	// Attaching projects is a second call against the policy created above. Terraform records no
	// state when Create reports an error, so anything that fails from here on has to take the new
	// policy with it — otherwise it survives untracked and the next apply creates a duplicate.
	if err := r.syncProjects(&plan, policyType, policyID); err != nil {
		r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error setting AppSec policy projects", err.Error())
		return
	}

	// Always re-read: create responses can echo an empty scm_posture scope even when the request
	// carried ids (backend soft-drops unknown/unlinked units). Read-back is also what surfaces a
	// successful project attach.
	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err == nil && instance == nil {
		err = fmt.Errorf("policy %s/%s could not be read back after create", policyType, policyID)
	}
	if err != nil {
		r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error reading AppSec policy after create", err.Error())
		return
	}
	if err := verifyScmPostureScopeApplied(&plan, instance); err != nil {
		r.rollbackCreatedPolicy(ctx, policyType, policyID, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error applying AppSec policy scope", err.Error())
		return
	}

	state := stateFromPlanAfterWrite(&plan, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// After a 504/timeout the backend may still be saving the policy, so the lookup is retried.
var ambiguousCreatePollDelays = []time.Duration{0, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}

var ambiguousCreateSleep = func(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// newSameNamePolicies polls until a policy absent from preexisting appears or the delays run out.
func (r *shiftLeftPolicyResource) newSameNamePolicies(ctx context.Context, policyType, name string, preexisting map[string]bool) ([]string, time.Duration, error) {
	var waited time.Duration
	var lastErr error
	for _, d := range ambiguousCreatePollDelays {
		if err := ambiguousCreateSleep(ctx, d); err != nil {
			return nil, waited, err
		}
		waited += d
		after, err := r.apiClient.ListShiftLeftPolicyIDsByName(policyType, name)
		if err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		var created []string
		for id := range after {
			if !preexisting[id] {
				created = append(created, id)
			}
		}
		if len(created) > 0 {
			sort.Strings(created)
			return created, waited, nil
		}
	}
	return nil, waited, lastErr
}

// createPolicy POSTs the policy. A POST with an ambiguous outcome (timeout / gateway error) may still
// have been committed; the client never replays it, so recover the id by diffing the same-name
// policies against the pre-create snapshot instead of leaving an untracked policy behind.
func (r *shiftLeftPolicyResource) createPolicy(ctx context.Context, policyType string, apiPolicy api_client.ShiftLeftPolicy, preexisting map[string]bool, diags *diag.Diagnostics) (string, error) {
	instance, err := r.apiClient.CreateShiftLeftPolicy(policyType, apiPolicy)
	if err == nil {
		return instance.ID, nil
	}
	if !errors.Is(err, api_client.ErrAmbiguousWrite) {
		return "", err
	}
	if preexisting == nil {
		return "", fmt.Errorf("%w; look for a %s policy named %q and import it before re-applying", err, policyType, apiPolicy.Name)
	}
	created, waited, listErr := r.newSameNamePolicies(ctx, policyType, apiPolicy.Name, preexisting)
	if listErr != nil {
		return "", fmt.Errorf("%w; checking whether it was created also failed (%v) — look for a %s policy named %q and import it before re-applying",
			err, listErr, policyType, apiPolicy.Name)
	}
	switch len(created) {
	case 0:
		return "", fmt.Errorf("%w; no new %s policy named %q appeared after %s — check the Orca UI before re-applying, and import it if it shows up",
			err, policyType, apiPolicy.Name, waited)
	case 1:
		diags.AddWarning("Recovered AppSec policy after an ambiguous create",
			fmt.Sprintf("The create request for %s policy %q failed without a definite answer (%v), but the policy was created as %s; it is now tracked in state.",
				policyType, apiPolicy.Name, err, created[0]))
		return created[0], nil
	default:
		return "", fmt.Errorf("%w; %d new %s policies named %q appeared (%s) — import the intended one and delete the rest",
			err, len(created), policyType, apiPolicy.Name, strings.Join(created, ", "))
	}
}

func (r *shiftLeftPolicyResource) rollbackCreatedPolicy(ctx context.Context, policyType, policyID string, diags *diag.Diagnostics) {
	if policyID == "" {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Rolling back AppSec policy %s/%s after a failed create", policyType, policyID))
	if err := r.deletePolicy(ctx, policyType, policyID); err != nil {
		diags.AddWarning(
			"Orphaned AppSec policy",
			fmt.Sprintf("Policy %s/%s was created but the apply failed afterwards, and deleting it during "+
				"rollback also failed: %s. It exists in Orca but not in Terraform state — delete it manually "+
				"or import it.", policyType, policyID, err.Error()),
		)
	}
}

const restorePriorPolicyWarning = "Failed to restore AppSec policy after a partial update"

// restorePriorPolicy rewrites the remote policy back to prior after a partial Update.
// restoreProjects is true when the projects endpoint already accepted the new set.
func (r *shiftLeftPolicyResource) restorePriorPolicy(ctx context.Context, prior *shiftLeftPolicyResourceModel, restoreProjects bool, diags *diag.Diagnostics) {
	if prior == nil || prior.ID.ValueString() == "" {
		return
	}
	policyType := prior.Type.ValueString()
	policyID := prior.ID.ValueString()
	tflog.Info(ctx, fmt.Sprintf("Restoring AppSec policy %s/%s after a partial update", policyType, policyID))

	apiPolicy, d := planToAPI(prior)
	if d.HasError() {
		diags.AddWarning(
			restorePriorPolicyWarning,
			fmt.Sprintf("Could not rebuild the prior policy body for %s/%s: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, d.Errors()),
		)
		return
	}
	apiPolicy.ProjectsIds = nil
	catalogDiags := diag.Diagnostics{}
	if !r.applyCatalog(prior, &apiPolicy, &catalogDiags) {
		diags.AddWarning(
			restorePriorPolicyWarning,
			fmt.Sprintf("Could not re-apply catalog enrichment for %s/%s: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, catalogDiags.Errors()),
		)
		return
	}
	if _, err := r.apiClient.UpdateShiftLeftPolicy(policyType, policyID, apiPolicy); err != nil {
		diags.AddWarning(
			restorePriorPolicyWarning,
			fmt.Sprintf("Restoring the prior body of %s/%s failed: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
				policyType, policyID, err.Error()),
		)
		return
	}
	if restoreProjects && tfconv.Known(prior.ProjectsIds) {
		restore := *prior
		restore.AttachAllProjects = types.BoolNull()
		if err := r.syncProjects(&restore, policyType, policyID); err != nil {
			diags.AddWarning(
				"Failed to restore AppSec policy projects after a partial update",
				fmt.Sprintf("Restoring projects on %s/%s failed: %s. The remote policy may not match Terraform state — run terraform refresh and reconcile manually.",
					policyType, policyID, err.Error()),
			)
		}
	}
}

func (r *shiftLeftPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyType := state.Type.ValueString()
	policyID := state.ID.ValueString()

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AppSec policy", fmt.Sprintf("Could not read policy %s/%s: %s", policyType, policyID, err.Error()))
		return
	}
	if instance == nil {
		tflog.Warn(ctx, fmt.Sprintf("AppSec policy %s/%s is missing on the remote side.", policyType, policyID))
		resp.State.RemoveResource(ctx)
		return
	}

	newState := apiToState(instance, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// syncProjects applies the plan's project attachment. attach_all_projects delegates the set to the
// API; otherwise a known list is sent verbatim and null/unknown leaves the attachment untouched.
// syncProjects is the only projects write (create, update and rollback all go through it).
func (r *shiftLeftPolicyResource) syncProjects(plan *shiftLeftPolicyResourceModel, policyType, policyID string) error {
	if projectsUnsupported(policyType) {
		return nil
	}
	if tfconv.BoolIsTrue(plan.AttachAllProjects) {
		return r.apiClient.AttachAllShiftLeftPolicyProjects(policyType, policyID)
	}
	if !tfconv.Known(plan.ProjectsIds) {
		return nil
	}
	return r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, tfconv.ListToStringSlice(plan.ProjectsIds))
}

// projectsIdsChanged is true when the attachment needs a write: attach_all_projects re-resolves the
// set server-side on every apply that reaches Update (ModifyPlan only plans one when it is stale),
// otherwise a known projects list that differs from state.
func projectsIdsChanged(plan, state *shiftLeftPolicyResourceModel) bool {
	if tfconv.BoolIsTrue(plan.AttachAllProjects) {
		return true
	}
	if !tfconv.Known(plan.ProjectsIds) {
		return false
	}
	return !plan.ProjectsIds.Equal(state.ProjectsIds)
}

func (r *shiftLeftPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Builtin.ValueBool() {
		if field, changed := builtinLockedFieldChanged(&plan, &state); changed {
			resp.Diagnostics.AddError(
				"Cannot modify built-in policy",
				fmt.Sprintf("Field %q is immutable on built-in Orca policies (the API locks it); other fields such as disabled, warn_mode, priority_failure_threshold, control overrides and projects_ids can be changed.", field),
			)
			return
		}
	}

	apiPolicy, diags := planToAPI(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiPolicy.ProjectsIds = nil // detach-all via dedicated endpoint, not main body omitempty.

	policyType := plan.Type.ValueString()
	policyID := plan.ID.ValueString()
	if !r.applyCatalog(&plan, &apiPolicy, &resp.Diagnostics) {
		return
	}

	_, err := r.apiClient.UpdateShiftLeftPolicy(policyType, policyID, apiPolicy)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AppSec policy", "Could not update policy: "+err.Error())
		return
	}

	// Sync projects only when known and actually changed; null/unknown means leave as-is.
	// SetShiftLeftPolicyProjects replaces the whole attachment set, so re-sending an
	// unchanged set is a needless detach/reattach on every apply.
	projectsChanged := projectsIdsChanged(&plan, &state)
	if projectsChanged {
		if err := r.syncProjects(&plan, policyType, policyID); err != nil {
			// Body PUT already landed; restore the prior body so live matches the still-current state.
			r.restorePriorPolicy(ctx, &state, false, &resp.Diagnostics)
			resp.Diagnostics.AddError("Error updating AppSec policy projects", err.Error())
			return
		}
	}

	instance, err := r.apiClient.GetShiftLeftPolicy(policyType, policyID)
	// The write already succeeded, so a missing policy here is a failed read-back, not a deletion.
	if err == nil && instance == nil {
		err = fmt.Errorf("policy %s/%s could not be read back after the update; run terraform refresh", policyType, policyID)
	}
	if err != nil {
		r.restorePriorPolicy(ctx, &state, projectsChanged, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error reading AppSec policy after update", err.Error())
		return
	}
	if err := verifyScmPostureScopeApplied(&plan, instance); err != nil {
		r.restorePriorPolicy(ctx, &state, projectsChanged, &resp.Diagnostics)
		resp.Diagnostics.AddError("Error applying AppSec policy scope", err.Error())
		return
	}

	newState := stateFromPlanAfterWrite(&plan, instance)
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *shiftLeftPolicyResource) applyCatalog(plan *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy, diags *diag.Diagnostics) bool {
	catalogType := policyTypeHandlers[plan.Type.ValueString()].catalogType
	if catalogType == "" {
		return true
	}
	if err := r.apiClient.AddAllCatalogControls(catalogType, apiPolicy, allControlsScopeKeys(plan)); err != nil {
		diags.AddError("Error expanding catalog controls", err.Error())
		return false
	}
	if err := r.apiClient.EnrichShiftLeftPolicyFromCatalog(catalogType, apiPolicy); err != nil {
		diags.AddError("Error enriching AppSec policy from catalog", err.Error())
		return false
	}
	return true
}

func (r *shiftLeftPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state shiftLeftPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Builtin.ValueBool() {
		resp.Diagnostics.AddError(
			"Cannot delete built-in policy",
			"Built-in Orca policies cannot be deleted via Terraform.",
		)
		return
	}

	policyType := state.Type.ValueString()
	policyID := state.ID.ValueString()

	if err := r.deletePolicy(ctx, policyType, policyID); err != nil {
		if isLastActivePolicyError(err) {
			resp.Diagnostics.AddError(
				"Cannot delete the last active policy of a project",
				"Orca refuses to leave a project without an active policy, and it rejects detaching the policy for "+
					"the same reason, so Terraform cannot resolve this on its own. Attach another active policy of "+
					"type "+policyType+" to the projects named below (or delete those projects) and re-run. "+
					"API response: "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError("Error deleting AppSec policy", "Could not delete policy: "+err.Error())
	}
}

// deletePolicy is the single way this resource removes a policy. Both the user's destroy and
// Create's rollback go through it, because the API's refusals are properties of deleting a policy,
// not of the destroy the user asked for: the last-active guard runs on every delete, and a policy
// already gone counts as deleted whoever is removing it.
func (r *shiftLeftPolicyResource) deletePolicy(ctx context.Context, policyType, policyID string) error {
	err := r.apiClient.DeleteShiftLeftPolicy(policyType, policyID)
	if err != nil && isLastActivePolicyError(err) {
		err = r.deleteAfterDetach(ctx, policyType, policyID, err)
	}
	if err == nil || strings.Contains(err.Error(), "404") {
		return nil
	}
	return err
}

func isLastActivePolicyError(err error) bool {
	return strings.Contains(err.Error(), "last active policy")
}

// deleteAfterDetach retries a delete the API refused as a project's last active policy, detaching
// the policy first. This bypasses nothing: the API applies the same guard to the detach, and rejects
// it whenever removing the policy would really leave a project unprotected. It succeeds only in the
// cases the API itself allows — most usefully a policy that is already disabled, which the delete
// guard flags even though a disabled policy is not what keeps a project active.
func (r *shiftLeftPolicyResource) deleteAfterDetach(ctx context.Context, policyType, policyID string, deleteErr error) error {
	tflog.Info(ctx, fmt.Sprintf("Delete of AppSec policy %s/%s was refused as a project's last active policy; retrying after a detach", policyType, policyID))
	if err := r.apiClient.SetShiftLeftPolicyProjects(policyType, policyID, nil); err != nil {
		return deleteErr // report the original refusal; the detach hit the same guard
	}
	return r.apiClient.DeleteShiftLeftPolicy(policyType, policyID)
}
