package shift_left_bitbucket_account

import (
	"context"
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = &bitbucketAccountResource{}
	_ resource.ResourceWithConfigure   = &bitbucketAccountResource{}
	_ resource.ResourceWithImportState = &bitbucketAccountResource{}
)

var bitbucketLabels = shift_left_integration.NewAdoptLabels("Bitbucket account")

type bitbucketAccountResource struct {
	apiClient *api_client.APIClient
}

func NewResource() resource.Resource { return &bitbucketAccountResource{} }

func (r *bitbucketAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_shift_left_bitbucket_account"
}

func (r *bitbucketAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.apiClient = shift_left_integration.ConfigureAPIClient(req)
}

func (r *bitbucketAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceSchema()
}

func (r *bitbucketAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shift_left_integration.ImportSlashPair(ctx, req, resp, "installation_id", "account_id", "<installation_id>/<account_id>")
}

func (r *bitbucketAccountResource) ops() shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel]{
		Labels: bitbucketLabels,
		UnitID: func(m *resourceModel) string { return m.AccountID.ValueString() },
		Get: func(m *resourceModel) (*api_client.BitbucketAccount, error) {
			return r.apiClient.GetBitbucketAccount(m.InstallationID.ValueString(), m.AccountID.ValueString())
		},
		Update: func(m *resourceModel, body api_client.ScmInstallationUpdate) (*api_client.BitbucketAccount, error) {
			return r.apiClient.UpdateBitbucketAccount(m.InstallationID.ValueString(), m.AccountID.ValueString(), body)
		},
		Snapshot: func(u *api_client.BitbucketAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Integrate the Orca Bitbucket account first, then import.",
		CreateErrorTitle: "Error configuring Bitbucket account",
		UpdateErrorTitle: "Error updating Bitbucket account",
	}
}

func (r *bitbucketAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.ops().DoCreate(ctx, req, resp)
}

func (r *bitbucketAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.ops().DoRead(ctx, req, resp)
}

func (r *bitbucketAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.ops().DoUpdate(ctx, req, resp)
}

func (r *bitbucketAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.ops().DoDelete(ctx, req, resp)
}
