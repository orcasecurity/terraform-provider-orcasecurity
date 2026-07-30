package shift_left_repository

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RepoConfigFields struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	URL                     types.String `tfsdk:"url"`
	Branch                  types.String `tfsdk:"branch"`
	ProjectID               types.String `tfsdk:"project_id"`
	Disabled                types.Bool   `tfsdk:"disabled"`
	DisableScanPullRequests types.Bool   `tfsdk:"disable_scan_pull_requests"`
	CommentsOnPullRequests  types.String `tfsdk:"comments_on_pull_requests"`
	PrSummaryComment        types.String `tfsdk:"pr_summary_comment"`
	SkipCheckRuns           types.String `tfsdk:"skip_check_runs"`
	ConfigFileSupport       types.String `tfsdk:"config_file_support"`
	Status                  types.String `tfsdk:"status"`
	RepositoryContextID     types.String `tfsdk:"repository_context_id"`
	IntegrationStatus       types.String `tfsdk:"integration_status"`
	ScmPosturePolicyID      types.String `tfsdk:"scm_posture_policy_id"`
}

func (f *RepoConfigFields) Fields() *RepoConfigFields { return f }

// GitHub/GitLab require branch on integrate (API 400); Azure/Bitbucket accept omitted branch.
func branchAttribute(branchRequired bool) rschema.StringAttribute {
	attr := rschema.StringAttribute{
		PlanModifiers: []planmodifier.String{branchRequiresReplace()},
	}
	// Create-only semantics and the import consequence are identical for all four SCMs; only
	// whether the API demands a branch on integrate differs.
	const createOnly = "Create-only: the API never returns or updates it, so Terraform cannot detect drift on " +
		"this attribute, and changing it re-integrates the repository — a destroy and create that deletes and " +
		"recreates its repository context. `terraform import` cannot read it back either, so an imported " +
		"repository has no branch in state and the first apply records the configured value in place, without " +
		"re-integrating."
	if branchRequired {
		attr.Required = true
		attr.Description = "Branch to scan. Required by this SCM's integration API. " + createOnly
	} else {
		attr.Optional = true
		attr.Description = "Branch to scan. Optional for this SCM: leave it unset to scan the repository default " +
			"branch, in which case no branch is ever stored and this attribute can never force a replacement. " +
			createOnly
	}
	return attr
}

// branchRequiresReplace forces re-integration when branch moves between two known values, but
// not when state holds no branch at all. The API never returns branch, so a freshly imported
// repository has a null branch while the config supplies one; an unconditional RequiresReplace
// reads that as a change and destroys and re-integrates every imported repository on its first
// apply, deleting its repository context. Falling through to Update instead is safe because
// fromAPI carries branch over from the plan, so state records the same value create would have.
func branchRequiresReplace() planmodifier.String {
	const description = "Changing the branch forces re-integration, except when state holds no branch " +
		"(a freshly imported repository), because the API never returns it."
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull()
		},
		description, description,
	)
}

func sharedRepoAttributes(traits providerTraits) map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:      true,
			Description:   "Orca id of the integrated repository row.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": rschema.StringAttribute{
			Required:      true,
			Description:   fmt.Sprintf("Repository name (path) as known to %s.", traits.name),
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"url": rschema.StringAttribute{
			Required:      true,
			Description:   "Repository URL.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"branch": branchAttribute(traits.branchRequired),
		"project_id": rschema.StringAttribute{
			Optional: true,
			Computed: true,
			Description: "Shift Left project to place the repository in. When omitted on create, Orca creates a dedicated " +
				"project for the repository. Changing it moves the repository between projects.",
		},
		"disabled": rschema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Pause scanning for this repository (the repository stays integrated).",
		},
		"disable_scan_pull_requests": rschema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Disable pull request scanning for this repository.",
		},
		"comments_on_pull_requests": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to comment on pull requests.",
			Validators:  shift_left_integration.PRCommentValidator(),
		},
		"pr_summary_comment": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to add a summary comment on pull requests.",
			Validators:  shift_left_integration.PRCommentValidator(),
		},
		"skip_check_runs": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to skip creating SCM check runs.",
			Validators:  shift_left_integration.SkipCheckRunsValidator(traits.skipCheckRunsValues),
		},
		"config_file_support": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the in-repo Orca config file is honored.",
			Validators:  shift_left_integration.ConfigFileSupportValidator(),
		},
		"status": rschema.StringAttribute{
			Computed:    true,
			Description: "Aggregated initial scan status.",
		},
		"repository_context_id": rschema.StringAttribute{
			Computed:    true,
			Description: "Repository context id; deleting this context is how the repository is un-integrated.",
		},
		"integration_status": rschema.StringAttribute{
			Computed:    true,
			Description: "Health status of the owning installation. Empty when healthy.",
		},
		"scm_posture_policy_id": rschema.StringAttribute{
			Computed:    true,
			Description: "SCM posture policy that applies to this repository, if any.",
		},
	}
}

type providerTraits struct {
	name                    string
	branchRequired          bool
	skipCheckRunsValues     []string
	skipCheckRunsUnreadable bool // Azure list omits skip_check_runs — preserve prior value to avoid phantom drift.
}

var (
	githubTraits = providerTraits{
		name:                "GitHub",
		branchRequired:      true,
		skipCheckRunsValues: shift_left_integration.FullSkipCheckRunValues,
	}
	gitlabTraits = providerTraits{
		name:                "GitLab",
		branchRequired:      true,
		skipCheckRunsValues: shift_left_integration.GitlabSkipCheckRunValues,
	}
	azureTraits = providerTraits{
		name:                    "Azure DevOps",
		skipCheckRunsValues:     shift_left_integration.FullSkipCheckRunValues,
		skipCheckRunsUnreadable: true,
	}
	bitbucketTraits = providerTraits{
		name:                "Bitbucket",
		skipCheckRunsValues: shift_left_integration.FullSkipCheckRunValues,
	}
)

func fromAPI(prior RepoConfigFields, api *api_client.ScmRepository, traits providerTraits) RepoConfigFields {
	out := RepoConfigFields{
		ID:                     types.StringValue(api.ID),
		Name:                   types.StringValue(api.RepositoryName),
		URL:                    types.StringValue(api.RepositoryURL),
		Branch:                 prior.Branch,
		ProjectID:              tfconv.StringOrNull(api.ProjectID),
		Disabled:               types.BoolValue(api.Disabled),
		CommentsOnPullRequests: tfconv.StringOrNull(api.CommentsOnPRs),
		PrSummaryComment:       tfconv.StringOrNull(api.PrSummaryComment),
		SkipCheckRuns:          tfconv.StringOrNull(api.SkipCheckRuns),
		ConfigFileSupport:      tfconv.StringOrNull(api.ConfigFileSupport),
		Status:                 tfconv.StringOrNull(api.Status),
		RepositoryContextID:    tfconv.StringOrNull(api.RepositoryContextID),
		IntegrationStatus:      tfconv.StringOrNull(api.IntegrationStatus),
		ScmPosturePolicyID:     tfconv.StringOrNull(api.ScmPosturePolicyID),
	}
	if api.DisableScanPRs != nil {
		out.DisableScanPullRequests = types.BoolValue(*api.DisableScanPRs)
	} else {
		out.DisableScanPullRequests = types.BoolNull()
	}
	if traits.skipCheckRunsUnreadable && api.SkipCheckRuns == "" && tfconv.Known(prior.SkipCheckRuns) {
		out.SkipCheckRuns = prior.SkipCheckRuns
	}
	return out
}

func planRepoConfig(plan *RepoConfigFields) (api_client.ScmRepoIntegrationConfig, bool) {
	cfg := api_client.ScmRepoIntegrationConfig{}
	set := false
	if tfconv.Known(plan.DisableScanPullRequests) {
		v := plan.DisableScanPullRequests.ValueBool()
		cfg.DisableScanPullRequests = &v
		set = true
	}
	setStr := func(dst *string, src types.String) {
		if tfconv.Known(src) {
			*dst = src.ValueString()
			set = true
		}
	}
	setStr(&cfg.CommentsOnPullRequests, plan.CommentsOnPullRequests)
	setStr(&cfg.PrSummaryComment, plan.PrSummaryComment)
	setStr(&cfg.SkipCheckRuns, plan.SkipCheckRuns)
	setStr(&cfg.ConfigFileSupport, plan.ConfigFileSupport)
	return cfg, set
}

func configUpdateBody(rowID string, plan *RepoConfigFields) (api_client.ScmRepositoryConfigUpdate, bool) {
	cfg, set := planRepoConfig(plan)
	body := api_client.ScmRepositoryConfigUpdate{IDs: []string{rowID}, ScmRepoIntegrationConfig: cfg}
	if tfconv.Known(plan.Disabled) {
		v := plan.Disabled.ValueBool()
		body.Disabled = &v
		set = true
	}
	return body, set
}

// disabled is not honored on integrate POST — apply via follow-up PATCH only.
func disabledUpdateBody(rowID string, plan *RepoConfigFields) (api_client.ScmRepositoryConfigUpdate, bool) {
	if !tfconv.Known(plan.Disabled) {
		return api_client.ScmRepositoryConfigUpdate{}, false
	}
	disabled := plan.Disabled.ValueBool()
	return api_client.ScmRepositoryConfigUpdate{IDs: []string{rowID}, Disabled: &disabled}, true
}

func integrateConfig(plan *RepoConfigFields) api_client.ScmRepoIntegrationConfig {
	cfg, _ := planRepoConfig(plan)
	return cfg
}

type repoOps struct {
	client       *api_client.APIClient
	traits       providerTraits
	integrate    func() error
	find         func() (*api_client.ScmRepository, error)
	update       func(api_client.ScmRepositoryConfigUpdate) error
	syncIdentity func(*api_client.ScmRepository) // Bitbucket slug backfill on read; nil elsewhere.
}

func repoCreate[M any](ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
	ops func(*M) repoOps, fields func(*M) *RepoConfigFields) {
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o := ops(&plan)
	row := createRepo(o, fields(&plan), &resp.Diagnostics)
	if row == nil {
		return
	}
	*fields(&plan) = fromAPI(*fields(&plan), row, o.traits)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func repoRead[M any](ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
	ops func(*M) repoOps, fields func(*M) *RepoConfigFields) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o := ops(&state)
	row, err := o.find()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading %s repository integration", o.traits.name), err.Error())
		return
	}
	if row == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	*fields(&state) = fromAPI(*fields(&state), row, o.traits)
	if o.syncIdentity != nil {
		o.syncIdentity(row)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func repoUpdate[M any](ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
	ops func(*M) repoOps, fields func(*M) *RepoConfigFields) {
	var plan, state M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o := ops(&plan)
	row := updateRepo(o, fields(&plan), fields(&state), &resp.Diagnostics)
	if row == nil {
		return
	}
	*fields(&plan) = fromAPI(*fields(&plan), row, o.traits)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func repoDelete[M any](ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
	ops func(*M) repoOps, fields func(*M) *RepoConfigFields) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteRepo(ops(&state), fields(&state), &resp.Diagnostics)
}

func createRepo(ops repoOps, plan *RepoConfigFields, diags *diag.Diagnostics) *api_client.ScmRepository {
	if err := ops.integrate(); err != nil {
		diags.AddError(fmt.Sprintf("Error integrating %s repository", ops.traits.name), err.Error())
		return nil
	}
	row, err := ops.find()
	if err == nil && row == nil {
		err = fmt.Errorf("repository not found after integration; verify the repository identifiers")
	}
	if err != nil {
		rollbackUnconfirmedIntegration(ops, diags)
		diags.AddError(fmt.Sprintf("Error reading %s repository after integration", ops.traits.name), err.Error())
		return nil
	}
	if body, set := disabledUpdateBody(row.ID, plan); set {
		if err := ops.update(body); err != nil {
			rollbackIntegration(ops, row, diags)
			diags.AddError(fmt.Sprintf("Error applying %s repository configuration", ops.traits.name), err.Error())
			return nil
		}
		reread, err := ops.find()
		if err == nil && reread == nil {
			err = fmt.Errorf("repository disappeared after configuration update")
		}
		if err != nil {
			rollbackIntegration(ops, row, diags)
			diags.AddError(fmt.Sprintf("Error re-reading %s repository", ops.traits.name), err.Error())
			return nil
		}
		row = reread
	}
	return row
}

// Delete repository_context on failed post-integrate config to avoid orphaned integrations.
func rollbackIntegration(ops repoOps, row *api_client.ScmRepository, diags *diag.Diagnostics) {
	ctxID := row.RepositoryContextID
	if ctxID == "" {
		return
	}
	if err := ops.client.DeleteRepositoryContext(ctxID); err != nil {
		diags.AddWarning(
			fmt.Sprintf("Failed to roll back %s repository integration", ops.traits.name),
			fmt.Sprintf("The repository was integrated but configuration failed, and removing repository context %s during rollback also failed: %s. Remove the integration manually or re-run to reconcile.", ctxID, err.Error()),
		)
	}
}

func rollbackUnconfirmedIntegration(ops repoOps, diags *diag.Diagnostics) {
	if row, err := ops.find(); err == nil && row != nil {
		rollbackIntegration(ops, row, diags)
		return
	}
	diags.AddWarning(
		fmt.Sprintf("Possible orphaned %s repository integration", ops.traits.name),
		"The integration request succeeded but the repository could not be read back, so it is not tracked in Terraform state. If a live integration persists, remove it manually or re-run to reconcile.",
	)
}

func updateRepo(ops repoOps, plan, state *RepoConfigFields, diags *diag.Diagnostics) *api_client.ScmRepository {
	// Move project before config PATCH — move failure must not leave remote config ahead of state.
	moveNeeded := tfconv.Known(plan.ProjectID) && plan.ProjectID.ValueString() != state.ProjectID.ValueString()
	if moveNeeded {
		ctxID := state.RepositoryContextID.ValueString()
		if ctxID == "" {
			diags.AddError(fmt.Sprintf("Error moving %s repository", ops.traits.name),
				"repository_context_id is unknown; run terraform refresh and retry")
			return nil
		}
		if err := ops.client.MoveRepositoryContexts(plan.ProjectID.ValueString(), []string{ctxID}); err != nil {
			diags.AddError(fmt.Sprintf("Error moving %s repository to project", ops.traits.name), err.Error())
			return nil
		}
	}
	if body, set := configUpdateBody(state.ID.ValueString(), plan); set {
		if err := ops.update(body); err != nil {
			diags.AddError(fmt.Sprintf("Error updating %s repository configuration", ops.traits.name), err.Error())
			return nil
		}
	}
	row, err := ops.find()
	if err == nil && row == nil {
		err = fmt.Errorf("repository disappeared during update")
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("Error re-reading %s repository", ops.traits.name), fmt.Sprintf("%v", err))
		return nil
	}
	return row
}

func deleteRepo(ops repoOps, state *RepoConfigFields, diags *diag.Diagnostics) {
	ctxID := state.RepositoryContextID.ValueString()
	if ctxID == "" {
		// Fall back to a live read; older state may predate the field.
		row, err := ops.find()
		if err != nil {
			diags.AddError(fmt.Sprintf("Error reading %s repository before delete", ops.traits.name), err.Error())
			return
		}
		if row == nil {
			return // already gone
		}
		ctxID = row.RepositoryContextID
		if ctxID == "" {
			diags.AddError(fmt.Sprintf("Error removing %s repository integration", ops.traits.name),
				"the repository has no repository_context_id; remove the integration manually")
			return
		}
	}
	if err := ops.client.DeleteRepositoryContext(ctxID); err != nil {
		diags.AddError(fmt.Sprintf("Error removing %s repository integration", ops.traits.name), err.Error())
	}
}
