package shift_left_repository

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// RepoSpec is the per-provider wiring for a repository resource built on the
// shared CRUD in repoCreate/repoRead/repoUpdate/repoDelete.
type RepoSpec[M any] struct {
	TypeNameSuffix string
	SchemaFn       func() rschema.Schema
	ImportFn       func(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse)
	Ops            func(*api_client.APIClient, *M) repoOps
	Fields         func(*M) *RepoConfigFields
}

// RepoResource shares Metadata/Configure/Schema/ImportState/CRUD across SCM
// repository types, mirroring shift_left_integration.GenericResource for units.
type RepoResource[M any] struct {
	Spec RepoSpec[M]

	apiClient *api_client.APIClient
}

func NewRepoResource[M any](spec RepoSpec[M]) resource.Resource {
	return &RepoResource[M]{Spec: spec}
}

var (
	_ resource.Resource                = &RepoResource[struct{}]{}
	_ resource.ResourceWithConfigure   = &RepoResource[struct{}]{}
	_ resource.ResourceWithImportState = &RepoResource[struct{}]{}
)

func (r *RepoResource[M]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.Spec.TypeNameSuffix
}

func (r *RepoResource[M]) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *RepoResource[M]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.Spec.SchemaFn()
}

func (r *RepoResource[M]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.Spec.ImportFn(ctx, req, resp)
}

func (r *RepoResource[M]) ops(m *M) repoOps {
	return r.Spec.Ops(r.apiClient, m)
}

func (r *RepoResource[M]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	repoCreate(ctx, req, resp, r.ops, r.Spec.Fields)
}

func (r *RepoResource[M]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	repoRead(ctx, req, resp, r.ops, r.Spec.Fields)
}

func (r *RepoResource[M]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	repoUpdate(ctx, req, resp, r.ops, r.Spec.Fields)
}

func (r *RepoResource[M]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	repoDelete(ctx, req, resp, r.ops, r.Spec.Fields)
}
