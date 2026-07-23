package shift_left_bitbucket_account

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_integration"

	"github.com/hashicorp/terraform-plugin-framework/path"
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
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "expected <installation_id>/<account_slug_or_orca_uuid>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("installation_id"), parts[0])...)
	if shift_left_integration.LooksLikeUUID(parts[1]) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[1])...)
}

func (r *bitbucketAccountResource) ops() shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel] {
	return shift_left_integration.AdoptedUnitOps[api_client.BitbucketAccount, resourceModel]{
		Labels: bitbucketLabels,
		UnitID: func(m *resourceModel) string {
			if m.ID.ValueString() != "" {
				return m.ID.ValueString()
			}
			return m.AccountID.ValueString()
		},
		Get: func(m *resourceModel) (*api_client.BitbucketAccount, error) {
			iid := m.InstallationID.ValueString()
			if id := m.ID.ValueString(); id != "" {
				return r.apiClient.GetBitbucketAccount(iid, id)
			}
			return r.apiClient.FindBitbucketAccountBySlug(iid, m.AccountID.ValueString())
		},
		Update: func(m *resourceModel, current *api_client.BitbucketAccount, body api_client.ScmInstallationUpdate) (*api_client.BitbucketAccount, error) {
			return r.apiClient.UpdateBitbucketAccount(m.InstallationID.ValueString(), current.ID, body)
		},
		Integrate: func(m *resourceModel, body api_client.ScmInstallationUpdate) error {
			return r.apiClient.IntegrateBitbucketUnit(api_client.BitbucketUnitIntegrate{
				InstallationID: m.InstallationID.ValueString(),
				AccountID:      m.AccountID.ValueString(),
				Body:           body,
			})
		},
		Delete: func(m *resourceModel) error {
			id := m.ID.ValueString()
			if id == "" {
				a, err := r.apiClient.FindBitbucketAccountBySlug(m.InstallationID.ValueString(), m.AccountID.ValueString())
				if err != nil {
					return err
				}
				if a == nil {
					return nil
				}
				id = a.ID
			}
			return r.apiClient.DeleteBitbucketAccount(m.InstallationID.ValueString(), id)
		},
		Snapshot: func(u *api_client.BitbucketAccount) shift_left_integration.ExistingUnit {
			return shift_left_integration.ExistingFromCommon(u.ScmUnitCommonFields)
		},
		ToState: apiToState,
		Config:  func(m *resourceModel) *shift_left_integration.ScmConfigFields { return &m.ScmConfigFields },
		Describe: func(m *resourceModel) string {
			return fmt.Sprintf("Account %q on installation %q", m.AccountID.ValueString(), m.InstallationID.ValueString())
		},
		CreateHint:       "Install the Orca Bitbucket parent connection first (orcasecurity_shift_left_bitbucket_installation).",
		CreateErrorTitle: "Error creating/configuring Bitbucket account",
		UpdateErrorTitle: "Error updating Bitbucket account",
		DeleteErrorTitle: "Error deleting Bitbucket account",
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
