package api_client

import "strings"

// SplitComplianceSection breaks a "/"-joined section path into the three fields
// the Orca API expects: category, sub_category and sub_sub_category. Any path
// deeper than three levels keeps the remainder in sub_sub_category, so that
// JoinComplianceSection reconstructs the original string exactly.
func SplitComplianceSection(section string) (category, subCategory, subSubCategory string) {
	parts := strings.SplitN(section, "/", 3)
	if len(parts) > 0 {
		category = parts[0]
	}
	if len(parts) > 1 {
		subCategory = parts[1]
	}
	if len(parts) > 2 {
		subSubCategory = parts[2]
	}
	return category, subCategory, subSubCategory
}

// JoinComplianceSection rebuilds the "/"-joined section path from the API's
// separate category fields, skipping empty levels.
func JoinComplianceSection(category, subCategory, subSubCategory string) string {
	parts := make([]string, 0, 3)
	for _, p := range []string{category, subCategory, subSubCategory} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}
