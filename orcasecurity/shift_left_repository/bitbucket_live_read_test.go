package shift_left_repository_test

import (
	"os"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/internal/acctest"
	"terraform-provider-orcasecurity/orcasecurity/shift_left_repository"

	"github.com/hashicorp/terraform-plugin-framework/path"
)

// TestAccBitbucketRepository_liveImportRead drives ImportState + Read directly against the real
// API, bypassing terraform apply/destroy entirely. It exercises bitbucketSyncSlug — the fix that
// backfills `slug` on Read because import can't set a Required+RequiresReplace attribute the API
// never accepts on the import path (see bitbucket.go). Nothing is ever written to Terraform state,
// so there is nothing for the test harness to tear down afterward.
func TestAccBitbucketRepository_liveImportRead(t *testing.T) {
	installationID := os.Getenv("ORCA_TEST_BB_INSTALLATION_ID")
	accountSlug := os.Getenv("ORCA_TEST_BB_ACCOUNT_SLUG")
	repositoryID := os.Getenv("ORCA_TEST_BB_REPOSITORY_ID")
	if installationID == "" || accountSlug == "" || repositoryID == "" {
		t.Skip("ORCA_TEST_BB_INSTALLATION_ID, ORCA_TEST_BB_ACCOUNT_SLUG and ORCA_TEST_BB_REPOSITORY_ID not set")
	}

	acctest.RunLiveImportRead(t, shift_left_repository.NewBitbucketRepositoryResource(),
		installationID+":"+accountSlug+":"+repositoryID,
		// id is assigned by Orca, so only its presence can be asserted.
		acctest.LiveAttrCheck{Path: path.Root("id")},
		acctest.LiveAttrCheck{Path: path.Root("installation_id"), Want: installationID},
		acctest.LiveAttrCheck{Path: path.Root("account_id"), Want: accountSlug},
		acctest.LiveAttrCheck{Path: path.Root("bitbucket_repository_id"), Want: repositoryID},
		// Import cannot set slug (Required+RequiresReplace); this proves Read backfilled it via
		// bitbucketSyncSlug instead of leaving it empty, which would plan a destroy/recreate.
		acctest.LiveAttrCheck{Path: path.Root("slug")},
	)
}
