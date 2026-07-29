package shift_left_integration

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// unitOps is the CRUD surface shared by AdoptedUnitOps and InstallationLifecycle.
type unitOps interface {
	DoCreate(context.Context, resource.CreateRequest, *resource.CreateResponse)
	DoRead(context.Context, resource.ReadRequest, *resource.ReadResponse)
	DoUpdate(context.Context, resource.UpdateRequest, *resource.UpdateResponse)
	DoDelete(context.Context, resource.DeleteRequest, *resource.DeleteResponse)
}

// GenericResource shares Metadata/Configure/Schema/ImportState/CRUD across SCM unit types.
type GenericResource[Ops unitOps] struct {
	TypeNameSuffix string
	SchemaFn       func() rschema.Schema
	ImportFn       func(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse)
	OpsFn          func(*api_client.APIClient) Ops

	client *api_client.APIClient
}

var (
	_ resource.Resource                = &GenericResource[AdoptedUnitOps[struct{}, struct{}]]{}
	_ resource.ResourceWithConfigure   = &GenericResource[AdoptedUnitOps[struct{}, struct{}]]{}
	_ resource.ResourceWithImportState = &GenericResource[AdoptedUnitOps[struct{}, struct{}]]{}
	_ resource.Resource                = &GenericResource[InstallationLifecycle[struct{}, struct{}]]{}
)

func (r *GenericResource[Ops]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.TypeNameSuffix
}

func (r *GenericResource[Ops]) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.client = ConfigureAPIClient(req)
}

func (r *GenericResource[Ops]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.SchemaFn()
}

func (r *GenericResource[Ops]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.ImportFn(ctx, req, resp)
}

func (r *GenericResource[Ops]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.OpsFn(r.client).DoCreate(ctx, req, resp)
}

func (r *GenericResource[Ops]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.OpsFn(r.client).DoRead(ctx, req, resp)
}

func (r *GenericResource[Ops]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.OpsFn(r.client).DoUpdate(ctx, req, resp)
}

func (r *GenericResource[Ops]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.OpsFn(r.client).DoDelete(ctx, req, resp)
}
