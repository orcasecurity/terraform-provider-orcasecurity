package shift_left_unit_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_unit"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

func TestAccBitbucketAccount_liveImportRead(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_BB_INSTALLATION_ID")
	accountSlug := os.Getenv("ORCA_TEST_BB_ACCOUNT_SLUG")
	if installationID == "" || accountSlug == "" {
		t.Skip("ORCA_TEST_BB_INSTALLATION_ID and ORCA_TEST_BB_ACCOUNT_SLUG not set")
	}

	acctest.RunLiveImportRead(t, shift_left_unit.NewBitbucketAccountResource(), installationID+"/"+accountSlug,
		// id is assigned by Orca, so only its presence can be asserted.
		acctest.LiveAttrCheck{Path: path.Root("id")},
		acctest.LiveAttrCheck{Path: path.Root("account_id"), Want: accountSlug},
		acctest.LiveAttrCheck{Path: path.Root("installation_id"), Want: installationID},
		acctest.LiveAttrCheck{Path: path.Root("configuration_settings").AtName("pr_summary_comment")},
	)
}
