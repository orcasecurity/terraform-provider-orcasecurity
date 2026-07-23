package shift_left_integration

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// InstallationBaseAttrs returns the schema attributes every SCM
// parent-installation resource (GitLab, Bitbucket, Azure DevOps) shares.
// Provider resources add their own token-detail attributes on top.
//   - scmName: display name, e.g. "GitLab".
//   - cloudURL: the SaaS URL used when server_url is omitted.
//   - tokenDesc: description of the access_token attribute.
func InstallationBaseAttrs(scmName, cloudURL, tokenDesc string) map[string]rschema.Attribute {
	return map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			Computed:      true,
			Description:   "Installation UUID.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": rschema.StringAttribute{
			Required:    true,
			Description: "Display name for the installation.",
		},
		"server_url": rschema.StringAttribute{
			Optional: true,
			Computed: true,
			Description: fmt.Sprintf("%s server URL without a trailing slash. Omit for %s cloud (%s).",
				scmName, scmName, cloudURL),
		},
		"access_token": rschema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: tokenDesc + " Write-only: never returned by the API.",
		},
		"external_server_url": rschema.StringAttribute{
			Computed:    true,
			Description: "Externally visible server URL, if different.",
		},
		"integration_status": rschema.StringAttribute{
			Computed:    true,
			Description: "Health status. Empty when healthy; `DISABLED_DUE_TO_INVALID_TOKEN` or `INSTALLATION_UNREACHABLE` otherwise.",
		},
		"cloud_integration": rschema.BoolAttribute{
			Computed:    true,
			Description: fmt.Sprintf("True when connected to %s cloud.", scmName),
		},
	}
}

// InstallationLifecycle wires provider-specific API calls and state mapping
// into the CRUD flow shared by the SCM parent-installation resources.
// M is the tfsdk resource model, A the api_client DTO.
type InstallationLifecycle[M any, A any] struct {
	// SCMName appears in error titles, e.g. "GitLab installation".
	SCMName string
	// Create POSTs the plan and returns the created row.
	Create func(plan *M) (*A, error)
	// Get reads by id; nil means the installation no longer exists.
	Get func(id string) (*A, error)
	// Update PATCHes the plan; nil (without error) means the row vanished.
	Update func(plan *M) (*A, error)
	// Delete removes the installation by id.
	Delete func(id string) error
	// ID extracts the installation id from the model.
	ID func(m *M) string
	// SetState copies API values onto the model, leaving write-only fields
	// (access_token) untouched.
	SetState func(m *M, a *A)
}

func (l InstallationLifecycle[M, A]) DoCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := l.Create(&plan)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error creating %s installation", l.SCMName), err.Error())
		return
	}
	l.SetState(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (l InstallationLifecycle[M, A]) DoRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	live, err := l.Get(l.ID(&state))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading %s installation", l.SCMName), err.Error())
		return
	}
	if live == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	l.SetState(&state, live) // access_token stays as-is: the API never returns it
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (l InstallationLifecycle[M, A]) DoUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	title := fmt.Sprintf("Error updating %s installation", l.SCMName)
	updated, err := l.Update(&plan)
	if err != nil {
		resp.Diagnostics.AddError(title, err.Error())
		return
	}
	if updated == nil {
		resp.Diagnostics.AddError(title, "installation disappeared after update")
		return
	}
	l.SetState(&plan, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (l InstallationLifecycle[M, A]) DoDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := l.Delete(l.ID(&state)); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error deleting %s installation", l.SCMName), err.Error())
	}
}
