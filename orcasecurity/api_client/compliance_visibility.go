package api_client

const (
	VisibilityPersonal       = "Personal"
	VisibilityOrganizational = "Organizational"

	// ErrPersonalOrg* match the API 400 for Personal + organization scope.
	ErrPersonalOrgSummary = "Personal framework cannot use organization scope"
	ErrPersonalOrgDetail  = "Personal frameworks can only be selected in user scope, not organization scope."

	// ErrVisibilityDowngrade* match the API 400 for Organizational → Personal.
	ErrVisibilityDowngradeSummary = "Cannot change visibility from Organizational to Personal"
	ErrVisibilityDowngradeDetail  = "Only personal frameworks can be promoted to organizational frameworks. Personal can be promoted to Organizational; the reverse is rejected by the API."
)

// PersonalRejectsOrganization is true when a Personal framework would be
// selected (or created) in organization scope. The API 400s that combination.
func PersonalRejectsOrganization(visibility *string, scopes []string) bool {
	if visibility == nil || *visibility != VisibilityPersonal {
		return false
	}
	for _, s := range scopes {
		if s == "organization" {
			return true
		}
	}
	return false
}

// VisibilityDowngrade is true for Organizational → Personal. The API accepts
// the other direction as a promotion.
func VisibilityDowngrade(from, to string) bool {
	return from == VisibilityOrganizational && to == VisibilityPersonal
}
