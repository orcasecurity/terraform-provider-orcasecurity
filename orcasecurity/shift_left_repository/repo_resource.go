package shift_left_repository

import (
	"context"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_common"

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
}

// RepoResource shares Metadata/Configure/Schema/ImportState/CRUD across SCM
// repository types, mirroring shift_left_integration.GenericResource for units.
// PM is always *M; it carries the proof that M embeds RepoConfigFields, which
// the shared CRUD uses to read the plan and write the API result back.
type RepoResource[M any, PM repoModelPtr[M]] struct {
	Spec RepoSpec[M]

	apiClient *api_client.APIClient
}

func NewRepoResource[M any, PM repoModelPtr[M]](spec RepoSpec[M]) resource.Resource {
	return &RepoResource[M, PM]{Spec: spec}
}

var (
	_ resource.Resource                = &RepoResource[githubRepositoryModel, *githubRepositoryModel]{}
	_ resource.ResourceWithConfigure   = &RepoResource[githubRepositoryModel, *githubRepositoryModel]{}
	_ resource.ResourceWithImportState = &RepoResource[githubRepositoryModel, *githubRepositoryModel]{}
)

func (r *RepoResource[M, PM]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.Spec.TypeNameSuffix
}

func (r *RepoResource[M, PM]) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_common.ConfigureAPIClient(req)
}

func (r *RepoResource[M, PM]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.Spec.SchemaFn()
}

func (r *RepoResource[M, PM]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.Spec.ImportFn(ctx, req, resp)
}

func (r *RepoResource[M, PM]) ops(m *M) repoOps {
	return r.Spec.Ops(r.apiClient, m)
}

func (r *RepoResource[M, PM]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	repoCreate[M, PM](ctx, req, resp, r.ops)
}

func (r *RepoResource[M, PM]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	repoRead[M, PM](ctx, req, resp, r.ops)
}

func (r *RepoResource[M, PM]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	repoUpdate[M, PM](ctx, req, resp, r.ops)
}

func (r *RepoResource[M, PM]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	repoDelete[M, PM](ctx, req, resp, r.ops)
}
