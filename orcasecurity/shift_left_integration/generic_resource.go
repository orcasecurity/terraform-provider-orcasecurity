package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// UnitOps is the CRUD surface shared by AdoptedUnitOps and InstallationLifecycle.
type UnitOps interface {
	DoCreate(context.Context, resource.CreateRequest, *resource.CreateResponse)
	DoRead(context.Context, resource.ReadRequest, *resource.ReadResponse)
	DoUpdate(context.Context, resource.UpdateRequest, *resource.UpdateResponse)
	DoDelete(context.Context, resource.DeleteRequest, *resource.DeleteResponse)
}

// planModifierOps is implemented by AdoptedUnitOps (not InstallationLifecycle).
type planModifierOps interface {
	ModifyPlan(context.Context, resource.ModifyPlanRequest, *resource.ModifyPlanResponse)
}

// GenericResource shares Metadata/Configure/Schema/ImportState/CRUD across SCM unit types.
type GenericResource struct {
	TypeNameSuffix string
	SchemaFn       func() rschema.Schema
	ImportFn       func(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse)
	OpsFn          func(*api_client.APIClient) UnitOps

	client *api_client.APIClient
}

var (
	_ resource.Resource                = &GenericResource{}
	_ resource.ResourceWithConfigure   = &GenericResource{}
	_ resource.ResourceWithImportState = &GenericResource{}
	_ resource.ResourceWithModifyPlan  = &GenericResource{}
)

func (r *GenericResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.TypeNameSuffix
}

func (r *GenericResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.client = ConfigureAPIClient(req)
}

func (r *GenericResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.SchemaFn()
}

func (r *GenericResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ImportFn(ctx, req, resp)
}

func (r *GenericResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.OpsFn(r.client).DoCreate(ctx, req, resp)
}

func (r *GenericResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.OpsFn(r.client).DoRead(ctx, req, resp)
}

func (r *GenericResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.OpsFn(r.client).DoUpdate(ctx, req, resp)
}

func (r *GenericResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.OpsFn(r.client).DoDelete(ctx, req, resp)
}

func (r *GenericResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if v, ok := r.OpsFn(r.client).(planModifierOps); ok {
		v.ModifyPlan(ctx, req, resp)
	}
}
