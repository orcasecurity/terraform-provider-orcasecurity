package shift_left_bitbucket_account_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_bitbucket_account"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

// TestAccBitbucketAccount_liveImportRead drives ImportState + Read directly
// against the real API, bypassing terraform apply/destroy entirely. Nothing is
// ever written to Terraform state, so there is nothing for the test harness to
// tear down afterward — no destroy call, no state rm needed, no lab mutation.
func TestAccBitbucketAccount_liveImportRead(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_BB_INSTALLATION_ID")
	accountSlug := os.Getenv("ORCA_TEST_BB_ACCOUNT_SLUG")
	if installationID == "" || accountSlug == "" {
		t.Skip("ORCA_TEST_BB_INSTALLATION_ID and ORCA_TEST_BB_ACCOUNT_SLUG not set")
	}

	acctest.RunLiveImportRead(t, shift_left_bitbucket_account.NewResource(), installationID+"/"+accountSlug,
		// id is assigned by Orca, so only its presence can be asserted.
		acctest.LiveAttrCheck{Path: path.Root("id")},
		acctest.LiveAttrCheck{Path: path.Root("account_id"), Want: accountSlug},
		acctest.LiveAttrCheck{Path: path.Root("installation_id"), Want: installationID},
		// Proves Read hydrated the nested config object, not just the identity attributes.
		acctest.LiveAttrCheck{Path: path.Root("configuration_settings").AtName("pr_summary_comment")},
	)
}
