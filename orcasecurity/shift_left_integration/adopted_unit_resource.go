package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// AdoptedUnitOps wires one provider's adopt-existing SCM unit resource
// (GitHub installation, GitLab group, Bitbucket / Azure DevOps account) into
// the shared CRUD flow. A is the api_client DTO; M is the tfsdk model
// embedding ScmConfigFields.
type AdoptedUnitOps[A any, M any] struct {
	Labels AdoptLabels
	// UnitID extracts the unit id from the model (used in read diagnostics).
	UnitID func(m *M) string
	// Get / Update run the provider API calls with identity from the model.
	Get    func(m *M) (*A, error)
	Update func(m *M, body api_client.ScmInstallationUpdate) (*A, error)
	// Snapshot extracts the adoptable fields from the live unit.
	Snapshot func(*A) ExistingUnit
	// ToState maps a live unit into a fresh model.
	ToState func(*A) M
	// Config exposes the model's embedded ScmConfigFields.
	Config func(*M) *ScmConfigFields
	// Describe renders the unit identity for not-found messages, e.g.
	// `Account "x" on installation "y"`.
	Describe func(m *M) string
	// CreateHint tells the user how to make the unit exist, e.g.
	// "Integrate the Orca Bitbucket account first, then import."
	CreateHint string
	// CreateErrorTitle / UpdateErrorTitle head the write diagnostics.
	CreateErrorTitle string
	UpdateErrorTitle string
}

func (o AdoptedUnitOps[A, M]) DoCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	o.write(ctx, req.Plan, req.Config, &resp.Diagnostics, &resp.State,
		func(m *M) string { return o.Describe(m) + " does not exist. " + o.CreateHint },
		o.CreateErrorTitle)
}

func (o AdoptedUnitOps[A, M]) DoUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	o.write(ctx, req.Plan, req.Config, &resp.Diagnostics, &resp.State,
		func(m *M) string { return o.Describe(m) + " was not found. It may have been removed; re-import." },
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
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// DoDelete only detaches state: integrated units cannot be deleted through
// these endpoints.
func (o AdoptedUnitOps[A, M]) DoDelete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	DeleteNoop(ctx, o.Labels)
}

// valueGetter abstracts tfsdk.Plan and tfsdk.Config for the shared write path.
type valueGetter interface {
	Get(ctx context.Context, target any) diag.Diagnostics
}

func (o AdoptedUnitOps[A, M]) write(
	ctx context.Context, planSrc, cfgSrc valueGetter,
	diags *diag.Diagnostics, state *tfsdk.State,
	notFound func(*M) string, title string,
) {
	var plan, config M
	diags.Append(planSrc.Get(ctx, &plan)...)
	diags.Append(cfgSrc.Get(ctx, &config)...)
	if diags.HasError() {
		return
	}
	planFields := o.Config(&plan)
	configFields := o.Config(&config)
	unit := AdoptWrite(diags, AdoptWriteRequest[A]{
		Get:             func() (*A, error) { return o.Get(&plan) },
		Update:          func(body api_client.ScmInstallationUpdate) (*A, error) { return o.Update(&plan, body) },
		Snapshot:        o.Snapshot,
		PlanMode:        planFields.InstallationMode,
		PlanDefault:     planFields.DefaultPolicies,
		PlanPolicies:    planFields.PoliciesIds,
		PlanConfig:      planFields.ConfigSettings,
		Project:         ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds, configFields.DefaultPolicies),
		Labels:          o.Labels,
		NotFoundMsg:     notFound(&plan),
		WriteErrorTitle: title,
	})
	if unit == nil {
		return
	}
	newState := o.ToState(unit)
	diags.Append(state.Set(ctx, &newState)...)
}
