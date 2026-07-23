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

// AdoptedUnitOps wires one provider's SCM unit resource (GitHub installation,
// GitLab group, Bitbucket / Azure DevOps account) into the shared CRUD flow.
// A is the api_client DTO; M is the tfsdk model embedding ScmConfigFields.
type AdoptedUnitOps[A any, M any] struct {
	Labels AdoptLabels
	// UnitID extracts the unit id from the model (used in read diagnostics).
	UnitID func(m *M) string
	// Get loads the live unit using state/plan identity (Orca UUID and/or SCM id).
	Get func(m *M) (*A, error)
	// Update PUTs the adopted body. current is the unit returned by Get (use its
	// Orca id — plan may not have id yet on first create-adopt).
	Update func(m *M, current *A, body api_client.ScmInstallationUpdate) (*A, error)
	// Integrate POSTs integrated_repositories to create a missing unit
	// (scan-all + empty repos). Nil means adopt-only (GitHub App must exist).
	Integrate func(m *M, body api_client.ScmInstallationUpdate) error
	// Delete tears down the live unit. Required for destroy = teardown.
	Delete func(m *M) error
	// Snapshot extracts the adoptable fields from the live unit.
	Snapshot func(*A) ExistingUnit
	// ToState maps a live unit into a fresh model.
	ToState func(*A) M
	// Config exposes the model's embedded ScmConfigFields.
	Config func(*M) *ScmConfigFields
	// Describe renders the unit identity for not-found messages.
	Describe func(m *M) string
	// CreateHint tells the user how to make the unit exist when Integrate is nil.
	CreateHint string
	// CreateErrorTitle / UpdateErrorTitle / DeleteErrorTitle head diagnostics.
	CreateErrorTitle string
	UpdateErrorTitle string
	DeleteErrorTitle string
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
		mode = types.StringValue("SCAN_ALL_INCLUDE_FUTURE")
	}
	if mode.ValueString() == "SELECTED_REPOSITORIES" {
		resp.Diagnostics.AddError(
			o.CreateErrorTitle,
			"Creating a missing unit with installation_mode=SELECTED_REPOSITORIES requires repository entries; "+
				"omit installation_mode (defaults to SCAN_ALL_INCLUDE_FUTURE) or set SCAN_ALL_INCLUDE_FUTURE, "+
				"or integrate repositories first and let this resource adopt the existing unit.",
		)
		return
	}

	ad := CreateUnitBody(mode, planFields.DefaultPolicies, planFields.PoliciesIds, planFields.ConfigSettings,
		ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds, configFields.DefaultPolicies))
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
		Project:         ProjectIntentFrom(configFields.ProjectID, configFields.PoliciesIds, configFields.DefaultPolicies),
		Labels:          o.Labels,
		NotFoundMsg:     notFoundMsg,
		WriteErrorTitle: title,
	})
	if unit == nil {
		return
	}
	newState := o.ToState(unit)
	diags.Append(state.Set(ctx, &newState)...)
}
