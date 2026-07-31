package shift_left_integration

import (
	"context"
	"errors"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type AdoptedUnitOps[A ScmUnit, M any] struct {
	Labels           AdoptLabels
	UnitID           func(m *M) string
	Get              func(m *M) (*A, error)
	Update           func(m *M, current *A, body api_client.ScmInstallationUpdate) (*A, error)
	Integrate        func(m *M, body api_client.ScmInstallationUpdate) error
	Delete           func(m *M) error
	ToState          func(*A) M
	Config           func(*M) *ScmConfigFields
	Describe         func(m *M) string
	CreateHint       string
	CreateErrorTitle string
	UpdateErrorTitle string
	DeleteErrorTitle string
}

// ModifyPlan rejects default_policies combinations that cannot round-trip after
// plan modifiers have applied (so carried-forward values are visible).
func (o AdoptedUnitOps[A, M]) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ValidateScmBindingPlan(o.Config(&plan), &resp.Diagnostics)
}

// Block create when unit already has repos unless adopt_existing=true — destroy would de-integrate out-of-band repos.
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
		repoCount := (*existing).Common().IntegratedRepositoriesCount
		if guardAdopt(repoCount, o.Config(&plan).AdoptExisting) {
			resp.Diagnostics.AddError("Refusing to adopt an already-integrated unit", adoptGuardDetail(o.Describe(&plan), repoCount))
			return
		}
		// Reuse unit fetched above instead of re-reading in writeAdopted.
		o.writeAdopted(ctx, &plan, &config, &resp.Diagnostics, &resp.State, writeAdoptedRequest[A]{
			NotFoundMsg: o.Describe(&plan) + " does not exist. " + o.CreateHint,
			Title:       o.CreateErrorTitle,
			Current:     existing,
		})
		return
	}

	if o.Integrate == nil {
		resp.Diagnostics.AddError(o.Labels.NotFoundTitle, o.Describe(&plan)+" does not exist. "+o.CreateHint)
		return
	}

	planFields := o.Config(&plan)
	configFields := o.Config(&config)

	body := CreateUnitBody(planFields.InstallationMode, planFields.DefaultPolicies, planFields.PoliciesIds,
		ConfigSettingsFromObject(planFields.ConfigSettings),
		ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds))
	if err := o.Integrate(&plan, body); err != nil {
		resp.Diagnostics.AddError(o.CreateErrorTitle, err.Error())
		return
	}

	created, err := o.Get(&plan)
	switch {
	case err != nil:
		o.rollbackIntegration(ctx, &plan, &resp.Diagnostics)
		resp.Diagnostics.AddError(o.CreateErrorTitle, err.Error())
		return
	case created == nil:
		o.rollbackIntegration(ctx, &plan, &resp.Diagnostics)
		resp.Diagnostics.AddError(o.Labels.NilReadTitle, o.Labels.NilReadDetail)
		return
	}
	o.setAdoptedState(ctx, &resp.Diagnostics, &resp.State, created, &plan)
}

// This path only runs for a unit that was not integrated before Create, so the integration is ours
// to undo. Terraform records no state when Create reports an error, so a live integration left
// behind here is invisible to Terraform: the next apply would try to integrate it again and hit the
// adopt guard instead, with nothing explaining why.
func (o AdoptedUnitOps[A, M]) rollbackIntegration(ctx context.Context, plan *M, diags *diag.Diagnostics) {
	orphanWarning := func(reason string) {
		diags.AddWarning(
			fmt.Sprintf("Possible orphaned %s integration", o.Describe(plan)),
			fmt.Sprintf("The integration succeeded but the unit could not be read back, so it is not tracked in "+
				"Terraform state, and %s. De-integrate it manually or import it to bring it under management.", reason),
		)
	}
	if o.Delete == nil {
		orphanWarning("this resource cannot de-integrate units")
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Rolling back integration of %s after a failed read-back", o.Describe(plan)))
	if err := o.Delete(plan); err != nil {
		if errors.Is(err, ErrUnitNotFound) {
			// DeleteByLookup re-ran the same lookup that just returned nil — the
			// integration is live but we have no id to tear it down with.
			orphanWarning("it could not be found again to de-integrate")
			return
		}
		orphanWarning("rolling it back also failed: " + err.Error())
	}
}

func (o AdoptedUnitOps[A, M]) DoUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, config M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o.writeAdopted(ctx, &plan, &config, &resp.Diagnostics, &resp.State, writeAdoptedRequest[A]{
		NotFoundMsg: o.Describe(&plan) + " was not found. It may have been removed; re-import.",
		Title:       o.UpdateErrorTitle,
	})
}

// setAdoptedState carries adopt_existing over from the prior plan or state (input-only, not
// on the API unit) and preserves values the API reports as absent but the caller set empty.
func (o AdoptedUnitOps[A, M]) setAdoptedState(ctx context.Context, diags *diag.Diagnostics, state *tfsdk.State, unit *A, prior *M) {
	newState := o.ToState(unit)
	priorFields := o.Config(prior)
	nextFields := o.Config(&newState)
	nextFields.AdoptExisting = priorFields.AdoptExisting
	preserveKnownEmpties(nextFields, priorFields)
	diags.Append(state.Set(ctx, &newState)...)
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
	o.setAdoptedState(ctx, &resp.Diagnostics, &resp.State, unit, &state)
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
		if errors.Is(err, ErrUnitNotFound) {
			// Already gone remotely — idempotent destroy.
			return
		}
		title := o.DeleteErrorTitle
		if title == "" {
			title = "Error deleting " + o.Describe(&state)
		}
		resp.Diagnostics.AddError(title, err.Error())
	}
}

type writeAdoptedRequest[A any] struct {
	NotFoundMsg string
	Title       string
	Current     *A // pre-fetched unit; nil means writeAdopted fetches it itself
}

// writeAdopted hydrates the plan against the live unit (Adopt) and PUTs the result.
func (o AdoptedUnitOps[A, M]) writeAdopted(
	ctx context.Context, plan, config *M,
	diags *diag.Diagnostics, state *tfsdk.State,
	req writeAdoptedRequest[A],
) {
	current := req.Current
	if current == nil {
		var err error
		if current, err = o.Get(plan); err != nil {
			diags.AddError(req.Title, err.Error())
			return
		}
	}
	if current == nil {
		diags.AddError(o.Labels.NotFoundTitle, req.NotFoundMsg)
		return
	}

	planFields := o.Config(plan)
	configFields := o.Config(config)
	body := Adopt(planFields.InstallationMode, planFields.DefaultPolicies, planFields.PoliciesIds,
		ConfigSettingsFromObject(planFields.ConfigSettings),
		ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds),
		(*current).Common())

	unit, err := o.Update(plan, current, body)
	switch {
	case err != nil:
		diags.AddError(req.Title, err.Error())
		return
	case unit == nil:
		diags.AddError(o.Labels.NilReadTitle, o.Labels.NilReadDetail)
		return
	}
	o.setAdoptedState(ctx, diags, state, unit, plan)
}
