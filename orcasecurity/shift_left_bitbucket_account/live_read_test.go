package shift_left_bitbucket_account_test

import (
	"context"
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity"
	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_bitbucket_account"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestAccBitbucketAccount_liveImportRead drives ImportState + Read directly
// against the real API, bypassing terraform apply/destroy entirely. Nothing is
// ever written to Terraform state, so there is nothing for the test harness to
// tear down afterward — no destroy call, no state rm needed, no lab mutation.
func TestAccBitbucketAccount_liveImportRead(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	installationID := os.Getenv("ORCA_TEST_BB_INSTALLATION_ID")
	accountSlug := os.Getenv("ORCA_TEST_BB_ACCOUNT_SLUG")
	if installationID == "" || accountSlug == "" {
		t.Skip("ORCA_TEST_BB_INSTALLATION_ID and ORCA_TEST_BB_ACCOUNT_SLUG not set")
	}
	orcasecurity.TestAccPreCheck(t)
	client := acctest.APIClient(t)

	ctx := context.Background()
	r := shift_left_bitbucket_account.NewResource()

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	sch := schemaResp.Schema

	configurable, ok := r.(resource.ResourceWithConfigure)
	if !ok {
		t.Fatal("resource does not implement ResourceWithConfigure")
	}
	var configResp resource.ConfigureResponse
	configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("configure: %v", configResp.Diagnostics)
	}

	importable, ok := r.(resource.ResourceWithImportState)
	if !ok {
		t.Fatal("resource does not implement ResourceWithImportState")
	}

	importResp := resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: sch,
			Raw:    tftypes.NewValue(sch.Type().TerraformType(ctx), nil),
		},
	}
	importable.ImportState(ctx, resource.ImportStateRequest{ID: installationID + "/" + accountSlug}, &importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("import state: %v", importResp.Diagnostics)
	}

	readResp := resource.ReadResponse{State: importResp.State}
	r.Read(ctx, resource.ReadRequest{State: importResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read: %v", readResp.Diagnostics)
	}

	var id, gotAccountID, gotInstallationID, prSummaryComment types.String
	if diags := readResp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
		t.Fatalf("get id: %v", diags)
	}
	if diags := readResp.State.GetAttribute(ctx, path.Root("account_id"), &gotAccountID); diags.HasError() {
		t.Fatalf("get account_id: %v", diags)
	}
	if diags := readResp.State.GetAttribute(ctx, path.Root("installation_id"), &gotInstallationID); diags.HasError() {
		t.Fatalf("get installation_id: %v", diags)
	}
	if diags := readResp.State.GetAttribute(ctx, path.Root("configuration_settings").AtName("pr_summary_comment"), &prSummaryComment); diags.HasError() {
		t.Fatalf("get configuration_settings.pr_summary_comment: %v", diags)
	}

	if id.IsNull() || id.ValueString() == "" {
		t.Error("id not populated by Read")
	}
	if gotAccountID.ValueString() != accountSlug {
		t.Errorf("account_id = %q, want %q", gotAccountID.ValueString(), accountSlug)
	}
	if gotInstallationID.ValueString() != installationID {
		t.Errorf("installation_id = %q, want %q", gotInstallationID.ValueString(), installationID)
	}
	if prSummaryComment.IsNull() {
		t.Error("configuration_settings.pr_summary_comment not populated by Read")
	}
}
