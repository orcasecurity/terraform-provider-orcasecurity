package add_users

import (
	"context"
	"terraform-provider-orcasecurity/orcasecurity/internal/testutils"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// basePlan returns a plan with every optional list null and email/role populated; each test
// overrides only the fields it probes.
func basePlan() addUsersResourceModel {
	return addUsersResourceModel{
		Email:             types.StringValue("tf-acc-test-x@example.com"),
		RoleID:            types.StringValue("role-1"),
		Groups:            types.ListNull(types.StringType),
		AllCloudAccounts:  types.BoolValue(false),
		CloudAccounts:     types.ListNull(types.StringType),
		UserFilters:       types.ListNull(types.StringType),
		ShiftleftProjects: types.ListNull(types.StringType),
		MFARequired:       types.BoolValue(false),
		ShouldSendEmail:   types.BoolValue(true),
	}
}

// modelToRequest must wrap the single email into a one-element list and copy every
// scalar and scoped list into the bulk_create payload.
func TestModelToRequest_MapsAllFields(t *testing.T) {
	r := &addUsersResource{}
	plan := basePlan()
	plan.AllCloudAccounts = types.BoolValue(true)
	plan.CloudAccounts = testutils.StringList(t, "ca-1", "ca-2")
	plan.UserFilters = testutils.StringList(t, "bu-1")
	plan.ShiftleftProjects = testutils.StringList(t, "sl-1", "sl-2")
	plan.MFARequired = types.BoolValue(true)
	plan.ShouldSendEmail = types.BoolValue(false)

	got, diags := r.modelToRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	// The single-invite resource sends the email as a one-element list.
	if len(got.InviteUserEmails) != 1 || got.InviteUserEmails[0] != "tf-acc-test-x@example.com" {
		t.Errorf("invite emails mismatch: %v", got.InviteUserEmails)
	}
	if got.RoleID != "role-1" {
		t.Errorf("role_id mismatch: %q", got.RoleID)
	}
	if !got.AllCloudAccounts || !got.MFARequired {
		t.Errorf("bool flags mismatch: all=%v mfa=%v", got.AllCloudAccounts, got.MFARequired)
	}
	if got.ShouldSendEmail {
		t.Error("should_send_email should be false")
	}
	if !testutils.SameElements(got.CloudAccounts, []string{"ca-1", "ca-2"}) {
		t.Errorf("cloud accounts mismatch: %v", got.CloudAccounts)
	}
	if !testutils.SameElements(got.UserFilters, []string{"bu-1"}) {
		t.Errorf("user filters mismatch: %v", got.UserFilters)
	}
	if !testutils.SameElements(got.ShiftleftProjects, []string{"sl-1", "sl-2"}) {
		t.Errorf("shiftleft mismatch: %v", got.ShiftleftProjects)
	}
}

// The groups path: a role_id-less invite must carry the group ids and an empty role.
func TestModelToRequest_GroupsPath(t *testing.T) {
	r := &addUsersResource{}
	plan := basePlan()
	plan.Email = types.StringValue("tf-acc-test-g@example.com")
	plan.RoleID = types.StringNull()
	plan.Groups = testutils.StringList(t, "grp-1", "grp-2")

	got, diags := r.modelToRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.RoleID != "" {
		t.Errorf("role_id should be empty on groups path, got %q", got.RoleID)
	}
	if !testutils.SameElements(got.Groups, []string{"grp-1", "grp-2"}) {
		t.Errorf("groups mismatch: %v", got.Groups)
	}
	if !got.ShouldSendEmail {
		t.Error("should_send_email should be true")
	}
}

// Null optional lists must become non-nil empty slices, so the payload sends
// explicit empty arrays rather than dropping the fields.
func TestModelToRequest_NullListsBecomeEmptySlices(t *testing.T) {
	r := &addUsersResource{}
	got, diags := r.modelToRequest(context.Background(), basePlan())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	for name, s := range map[string][]string{
		"groups":             got.Groups,
		"cloud_accounts":     got.CloudAccounts,
		"user_filters":       got.UserFilters,
		"shiftleft_projects": got.ShiftleftProjects,
	} {
		if s == nil {
			t.Errorf("%s should be non-nil empty slice, got nil", name)
		}
		if len(s) != 0 {
			t.Errorf("%s should be empty, got %v", name, s)
		}
	}
}

// An unknown email (as during a plan with a computed source) must still produce a
// single-element email list, using the empty string rather than panicking.
func TestModelToRequest_NullEmailProducesSingleEntry(t *testing.T) {
	r := &addUsersResource{}
	plan := basePlan()
	plan.Email = types.StringNull()

	got, diags := r.modelToRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(got.InviteUserEmails) != 1 || got.InviteUserEmails[0] != "" {
		t.Errorf("expected single empty-string email, got %v", got.InviteUserEmails)
	}
}
