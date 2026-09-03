package api_client

import "testing"

func TestPersonalRejectsOrganization(t *testing.T) {
	personal := VisibilityPersonal
	org := VisibilityOrganizational
	if PersonalRejectsOrganization(nil, []string{ScopeOrganization}) {
		t.Fatal("nil visibility must not reject")
	}
	if PersonalRejectsOrganization(&org, []string{ScopeOrganization}) {
		t.Fatal("Organizational + organization is legal")
	}
	if PersonalRejectsOrganization(&personal, []string{ScopeUser}) {
		t.Fatal("Personal + user is legal")
	}
	if !PersonalRejectsOrganization(&personal, []string{ScopeOrganization}) {
		t.Fatal("Personal + organization must reject")
	}
}

func TestVisibilityDowngrade(t *testing.T) {
	if !VisibilityDowngrade(VisibilityOrganizational, VisibilityPersonal) {
		t.Fatal("Organizational → Personal is a downgrade")
	}
	if VisibilityDowngrade(VisibilityPersonal, VisibilityOrganizational) {
		t.Fatal("Personal → Organizational is a promotion")
	}
}
