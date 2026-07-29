package shift_left_integration

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AdoptedUnitOps[A any, M any] struct {
	Labels           AdoptLabels
	UnitID           func(m *M) string
	Get              func(m *M) (*A, error)
	Update           func(m *M, current *A, body api_client.ScmInstallationUpdate) (*A, error)
	Integrate        func(m *M, body api_client.ScmInstallationUpdate) error
	Delete           func(m *M) error
	Snapshot         func(*A) ExistingUnit
	ToState          func(*A) M
	Config           func(*M) *ScmConfigFields
	Describe         func(m *M) string
	CreateHint       string
	CreateErrorTitle string
	UpdateErrorTitle string
	DeleteErrorTitle string
}

// guardAdopt reports whether Create must refuse to silently take over a unit
// that is already integrated in Orca. These resources adopt a pre-existing SCM
// unit rather than create one; adopting a unit that already has integrated
// repositories and later destroying it would remove repositories configured
// outside Terraform, so that case requires an explicit adopt_existing opt-in.
func guardAdopt(repoCount int64, adoptExisting types.Bool) bool {
	return repoCount > 0 && !adoptExisting.ValueBool()
}

func adoptGuardDetail(describe string, repoCount int64) string {
	return fmt.Sprintf(
		"%s is already integrated in Orca with %d integrated repositor%s that may have been configured outside this Terraform resource. "+
			"Applying would take over that integration, and a later `terraform destroy` would DE-INTEGRATE it — removing those repositories and their settings. "+
			"To bring this unit under management without a takeover write, import it instead of applying (terraform import). "+
			"If you intend to manage (and eventually tear down) an integration you did not create here, set `adopt_existing = true`.",
		describe, repoCount, repositoryPlural(repoCount))
}

func repositoryPlural(n int64) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func (o AdoptedUnitOps[A, M]) DoCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, config M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	existing, err := o.Get(&plan)
	if err != nil {
		resp.Diagnostics.AddError(o.CreateErrorTitle, err.Error())
		return
	}
	if existing != nil {
		if guardAdopt(o.Snapshot(existing).RepoCount, o.Config(&plan).AdoptExisting) {
			resp.Diagnostics.AddError("Refusing to adopt an already-integrated unit", adoptGuardDetail(o.Describe(&plan), o.Snapshot(existing).RepoCount))
			return
		}
		o.writeAdopted(ctx, &plan, &config, &resp.Diagnostics, &resp.State,
			o.Describe(&plan)+" does not exist. "+o.CreateHint, o.CreateErrorTitle)
		return
	}

	if o.Integrate == nil {
		resp.Diagnostics.AddError(o.Labels.NotFoundTitle, o.Describe(&plan)+" does not exist. "+o.CreateHint)
		return
	}

	planFields := o.Config(&plan)
	configFields := o.Config(&config)
	mode := planFields.InstallationMode
	if mode.IsNull() || mode.IsUnknown() {
		// Match the API and UI default. SCAN_ALL_INCLUDE_FUTURE would silently
		// enroll every current and future repository in the org/group for
		// scanning; SELECTED_REPOSITORIES integrates the unit and scans nothing
		// until repositories are added explicitly.
		mode = types.StringValue("SELECTED_REPOSITORIES")
	}

	ad := CreateUnitBody(mode, planFields.DefaultPolicies, planFields.PoliciesIds, planFields.ConfigSettings,
		ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds))
	if err := o.Integrate(&plan, ad.Body); err != nil {
		resp.Diagnostics.AddError(o.CreateErrorTitle, err.Error())
		return
	}

	created, err := o.Get(&plan)
	if err != nil {
		resp.Diagnostics.AddError(o.CreateErrorTitle, err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError(o.Labels.NilReadTitle, o.Labels.NilReadDetail)
		return
	}
	newState := o.ToState(created)
	o.Config(&newState).AdoptExisting = o.Config(&plan).AdoptExisting
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (o AdoptedUnitOps[A, M]) DoUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o.writeAdopted(ctx, &plan, &config, &resp.Diagnostics, &resp.State,
		o.Describe(&plan)+" was not found. It may have been removed; re-import.",
		o.UpdateErrorTitle)
}

func (o AdoptedUnitOps[A, M]) DoRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	unit := ReadUnit(ctx, &resp.Diagnostics, o.Labels, o.UnitID(&state),
		func() (*A, error) { return o.Get(&state) },
		resp.State.RemoveResource,
	)
	if unit == nil {
		return
	}
	newState := o.ToState(unit)
	o.Config(&newState).AdoptExisting = o.Config(&state).AdoptExisting
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (o AdoptedUnitOps[A, M]) DoDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if o.Delete == nil {
		DeleteNoop(ctx, o.Labels)
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Deleting live %s", o.Describe(&state)))
	if err := o.Delete(&state); err != nil {
		title := o.DeleteErrorTitle
		if title == "" {
			title = "Error deleting " + o.Describe(&state)
		}
		resp.Diagnostics.AddError(title, err.Error())
	}
}

func (o AdoptedUnitOps[A, M]) writeAdopted(
	ctx context.Context, plan, config *M,
	diags *diag.Diagnostics, state *tfsdk.State,
	notFoundMsg, title string,
) {
	planFields := o.Config(plan)
	configFields := o.Config(config)
	unit := AdoptWrite(diags, AdoptWriteRequest[A]{
		Get: func() (*A, error) { return o.Get(plan) },
		Update: func(current *A, body api_client.ScmInstallationUpdate) (*A, error) {
			return o.Update(plan, current, body)
		},
		Snapshot:        o.Snapshot,
		PlanMode:        planFields.InstallationMode,
		PlanDefault:     planFields.DefaultPolicies,
		PlanPolicies:    planFields.PoliciesIds,
		PlanConfig:      planFields.ConfigSettings,
		Project:         ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds),
		Labels:          o.Labels,
		NotFoundMsg:     notFoundMsg,
		WriteErrorTitle: title,
	})
	if unit == nil {
		return
	}
	newState := o.ToState(unit)
	o.Config(&newState).AdoptExisting = o.Config(plan).AdoptExisting
	diags.Append(state.Set(ctx, &newState)...)
}
