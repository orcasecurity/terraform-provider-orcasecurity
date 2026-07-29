package shift_left_integration

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

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

type InstallationLifecycle[M any, A any] struct {
	SCMName  string
	Create   func(plan *M) (*A, error)
	Get      func(id string) (*A, error)
	Update   func(plan *M) (*A, error)
	Delete   func(id string) error
	ID       func(m *M) string
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

type InstallationResource[M any, A any] struct {
	TypeNameSuffix string
	SchemaFn       func() rschema.Schema
	ImportFn       func(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse)
	LifecycleFn    func(*api_client.APIClient) InstallationLifecycle[M, A]

	client *api_client.APIClient
}

var (
	_ resource.Resource                = &InstallationResource[struct{}, struct{}]{}
	_ resource.ResourceWithConfigure   = &InstallationResource[struct{}, struct{}]{}
	_ resource.ResourceWithImportState = &InstallationResource[struct{}, struct{}]{}
)

func (r *InstallationResource[M, A]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.TypeNameSuffix
}

func (r *InstallationResource[M, A]) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.client = ConfigureAPIClient(req)
}

func (r *InstallationResource[M, A]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.SchemaFn()
}

func (r *InstallationResource[M, A]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ImportFn(ctx, req, resp)
}

func (r *InstallationResource[M, A]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.LifecycleFn(r.client).DoCreate(ctx, req, resp)
}

func (r *InstallationResource[M, A]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.LifecycleFn(r.client).DoRead(ctx, req, resp)
}

func (r *InstallationResource[M, A]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.LifecycleFn(r.client).DoUpdate(ctx, req, resp)
}

func (r *InstallationResource[M, A]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.LifecycleFn(r.client).DoDelete(ctx, req, resp)
}
