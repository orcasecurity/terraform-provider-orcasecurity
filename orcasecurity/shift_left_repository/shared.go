// Package shift_left_repository implements the per-repository integration
// resources (orcasecurity_shift_left_{github,gitlab,bitbucket,azure_devops}_repository).
//
// All four share the same lifecycle against different provider endpoints:
// create POSTs to {provider}/integrated_repositories/ (empty response, so the
// created row is re-found by its SCM-side id), config updates go through the
// bulk PATCH with a single-id list, project moves through
// repository_contexts/move_project/, and delete removes the repository
// context (the integrated_repositories endpoints define no DELETE).
package shift_left_repository

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RepoConfigFields are the shared Terraform attributes every repository
// resource carries beyond its provider-specific identity keys.
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

// sharedRepoAttributes builds the shared attribute map. skipCheckRunsValues
// differs per provider (GitLab only supports ALWAYS/NEVER).
func sharedRepoAttributes(scmName string, skipCheckRunsValues []string) map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:      true,
			Description:   "Orca id of the integrated repository row.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": rschema.StringAttribute{
			Required:      true,
			Description:   fmt.Sprintf("Repository name (path) as known to %s.", scmName),
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"url": rschema.StringAttribute{
			Required:      true,
			Description:   "Repository URL.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"branch": rschema.StringAttribute{
			Optional: true,
			Description: "Branch to scan. Omit for the repository default branch. Create-only: the API neither returns " +
				"nor updates it after integration, so changing it forces re-integration.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
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
			Validators: []validator.String{
				stringvalidator.OneOf("ALWAYS", "ONLY_ON_FAILED_ISSUES", "NEVER"),
			},
		},
		"pr_summary_comment": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to add a summary comment on pull requests.",
			Validators: []validator.String{
				stringvalidator.OneOf("ALWAYS", "ONLY_ON_FAILED_ISSUES", "NEVER"),
			},
		},
		"skip_check_runs": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "When to skip creating SCM check runs.",
			Validators: []validator.String{
				stringvalidator.OneOf(skipCheckRunsValues...),
			},
		},
		"config_file_support": rschema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "Whether the in-repo Orca config file is honored.",
			Validators: []validator.String{
				stringvalidator.OneOf("ENABLED", "DISABLED"),
			},
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

var fullSkipCheckRuns = []string{"ALWAYS", "ONLY_ON_INTERNAL_ISSUE", "NEVER"}
var gitlabSkipCheckRuns = []string{"ALWAYS", "NEVER"}

// fromAPI refreshes the shared fields from a live row. prior supplies values
// the API never echoes (branch always; skip_check_runs and
// scm_posture_policy_id on providers whose list serializer omits them).
func fromAPI(prior RepoConfigFields, api *api_client.ScmRepository) RepoConfigFields {
	out := RepoConfigFields{
		ID:                     types.StringValue(api.ID),
		Name:                   types.StringValue(api.RepositoryName),
		URL:                    types.StringValue(api.RepositoryURL),
		Branch:                 prior.Branch,
		ProjectID:              shift_left_integration.OptionalID(api.ProjectID),
		Disabled:               types.BoolValue(api.Disabled),
		CommentsOnPullRequests: shift_left_integration.OptionalID(api.CommentsOnPRs),
		PrSummaryComment:       shift_left_integration.OptionalID(api.PrSummaryComment),
		SkipCheckRuns:          shift_left_integration.OptionalID(api.SkipCheckRuns),
		ConfigFileSupport:      shift_left_integration.OptionalID(api.ConfigFileSupport),
		Status:                 shift_left_integration.OptionalID(api.Status),
		RepositoryContextID:    shift_left_integration.OptionalID(api.RepositoryContextID),
		IntegrationStatus:      shift_left_integration.OptionalID(api.IntegrationStatus),
		ScmPosturePolicyID:     shift_left_integration.OptionalID(api.ScmPosturePolicyID),
	}
	if api.DisableScanPRs != nil {
		out.DisableScanPullRequests = types.BoolValue(*api.DisableScanPRs)
	} else {
		out.DisableScanPullRequests = types.BoolNull()
	}
	// Azure's list serializer omits skip_check_runs entirely; keep the last
	// written value instead of flapping to null.
	if api.SkipCheckRuns == "" && !prior.SkipCheckRuns.IsNull() && !prior.SkipCheckRuns.IsUnknown() {
		out.SkipCheckRuns = prior.SkipCheckRuns
	}
	return out
}

// configUpdateBody assembles the bulk PATCH body for the set config fields.
// Returns false when nothing is set (PATCH can be skipped).
func configUpdateBody(rowID string, plan *RepoConfigFields) (api_client.ScmRepositoryConfigUpdate, bool) {
	body := api_client.ScmRepositoryConfigUpdate{IDs: []string{rowID}}
	set := false
	if known(plan.Disabled) {
		v := plan.Disabled.ValueBool()
		body.Disabled = &v
		set = true
	}
	if known(plan.DisableScanPullRequests) {
		v := plan.DisableScanPullRequests.ValueBool()
		body.DisableScanPullRequests = &v
		set = true
	}
	if known(plan.CommentsOnPullRequests) {
		body.CommentsOnPullRequests = plan.CommentsOnPullRequests.ValueString()
		set = true
	}
	if known(plan.PrSummaryComment) {
		body.PrSummaryComment = plan.PrSummaryComment.ValueString()
		set = true
	}
	if known(plan.SkipCheckRuns) {
		body.SkipCheckRuns = plan.SkipCheckRuns.ValueString()
		set = true
	}
	if known(plan.ConfigFileSupport) {
		body.ConfigFileSupport = plan.ConfigFileSupport.ValueString()
		set = true
	}
	return body, set
}

func known(v interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// repoOps are the provider-specific operations behind the shared lifecycle.
type repoOps struct {
	scmName   string
	integrate func() error
	find      func() (*api_client.ScmRepository, error)
	update    func(api_client.ScmRepositoryConfigUpdate) error
}

// createRepo runs the shared create flow: integrate, re-find the created row,
// apply config overrides, and return the refreshed row.
func createRepo(ops repoOps, plan *RepoConfigFields, diags *diag.Diagnostics) *api_client.ScmRepository {
	if err := ops.integrate(); err != nil {
		diags.AddError(fmt.Sprintf("Error integrating %s repository", ops.scmName), err.Error())
		return nil
	}
	row, err := ops.find()
	if err == nil && row == nil {
		err = fmt.Errorf("repository not found after integration; verify the repository identifiers")
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("Error reading %s repository after integration", ops.scmName), err.Error())
		return nil
	}
	if body, set := configUpdateBody(row.ID, plan); set {
		if err := ops.update(body); err != nil {
			diags.AddError(fmt.Sprintf("Error applying %s repository configuration", ops.scmName), err.Error())
			return nil
		}
		row, err = ops.find()
		if err != nil || row == nil {
			diags.AddError(fmt.Sprintf("Error re-reading %s repository", ops.scmName), fmt.Sprintf("%v", err))
			return nil
		}
	}
	return row
}

// updateRepo runs the shared update flow: config PATCH plus an optional
// project move, returning the refreshed row.
func updateRepo(client *api_client.APIClient, ops repoOps, plan, state *RepoConfigFields, diags *diag.Diagnostics) *api_client.ScmRepository {
	if body, set := configUpdateBody(state.ID.ValueString(), plan); set {
		if err := ops.update(body); err != nil {
			diags.AddError(fmt.Sprintf("Error updating %s repository configuration", ops.scmName), err.Error())
			return nil
		}
	}
	if known(plan.ProjectID) && plan.ProjectID.ValueString() != state.ProjectID.ValueString() {
		ctxID := state.RepositoryContextID.ValueString()
		if ctxID == "" {
			diags.AddError(fmt.Sprintf("Error moving %s repository", ops.scmName),
				"repository_context_id is unknown; run terraform refresh and retry")
			return nil
		}
		if err := client.MoveRepositoryContexts(plan.ProjectID.ValueString(), []string{ctxID}); err != nil {
			diags.AddError(fmt.Sprintf("Error moving %s repository to project", ops.scmName), err.Error())
			return nil
		}
	}
	row, err := ops.find()
	if err == nil && row == nil {
		err = fmt.Errorf("repository disappeared during update")
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("Error re-reading %s repository", ops.scmName), fmt.Sprintf("%v", err))
		return nil
	}
	return row
}

// deleteRepo un-integrates by deleting the repository context.
func deleteRepo(client *api_client.APIClient, ops repoOps, state *RepoConfigFields, diags *diag.Diagnostics) {
	ctxID := state.RepositoryContextID.ValueString()
	if ctxID == "" {
		// Fall back to a live read; older state may predate the field.
		row, err := ops.find()
		if err != nil {
			diags.AddError(fmt.Sprintf("Error reading %s repository before delete", ops.scmName), err.Error())
			return
		}
		if row == nil {
			return // already gone
		}
		ctxID = row.RepositoryContextID
	}
	if err := client.DeleteRepositoryContext(ctxID); err != nil {
		diags.AddError(fmt.Sprintf("Error removing %s repository integration", ops.scmName), err.Error())
	}
}
