package custom_compliance_framework

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type testModel struct {
	RuleID            types.String `tfsdk:"rule_id"`
	RuleIDInFramework types.String `tfsdk:"rule_id_in_framework"`
	Priority          types.String `tfsdk:"priority"`
	ControlUniqueID   types.String `tfsdk:"control_unique_id"`
	OriginFrameworkID types.String `tfsdk:"origin_framework_id"`
	ReferenceID       types.String `tfsdk:"reference_id"`
}

// tooDeepSectionModel is the fourth nesting level. The schema accepts it so
// ValidateConfig can reject it with a data-loss message; the API would 200 and
// silently reparent its controls onto the third level.
type tooDeepSectionModel struct {
	Name  types.String `tfsdk:"name"`
	Tests []testModel  `tfsdk:"tests"`
}

type leafSectionModel struct {
	Name     types.String          `tfsdk:"name"`
	Tests    []testModel           `tfsdk:"tests"`
	Sections []tooDeepSectionModel `tfsdk:"sections"`
}

type midSectionModel struct {
	Name     types.String       `tfsdk:"name"`
	Tests    []testModel        `tfsdk:"tests"`
	Sections []leafSectionModel `tfsdk:"sections"`
}

type sectionModel struct {
	Name     types.String      `tfsdk:"name"`
	Tests    []testModel       `tfsdk:"tests"`
	Sections []midSectionModel `tfsdk:"sections"`
}

const mixedSectionMessage = "a section cannot have both tests and sub-sections; the API would silently flatten it and the sub-section would inherit the parent name"

const depthSectionMessage = "nesting is at most three levels; the API would silently drop a fourth level and reparent its controls onto the third"

const invalidSectionSummary = "Invalid section"

func sectionHasTestsAndChildren(tests, children int) bool {
	return tests > 0 && children > 0
}

func mixedSectionError(path string) string {
	return fmt.Sprintf("%s: %s", path, mixedSectionMessage)
}

// DeriveRuleIDInFramework matches the UI: `${section.id}.${index+1}`.
func DeriveRuleIDInFramework(sectionID string, index int) string {
	return fmt.Sprintf("%s.%d", sectionID, index+1)
}

func positionalSectionID(parentID string, index int) string {
	n := fmt.Sprintf("%d", index+1)
	if parentID == "" {
		return n
	}
	return parentID + "." + n
}

func testsToAPI(sectionID string, tests []testModel) []api_client.CustomComplianceFrameworkTest {
	out := make([]api_client.CustomComplianceFrameworkTest, 0, len(tests))
	for i, t := range tests {
		rid := t.RuleIDInFramework.ValueString()
		if rid == "" {
			rid = DeriveRuleIDInFramework(sectionID, i)
		}
		out = append(out, api_client.CustomComplianceFrameworkTest{
			RuleID:            t.RuleID.ValueString(),
			RuleIDInFramework: rid,
			Priority:          t.Priority.ValueString(),
			ControlUniqueID:   t.ControlUniqueID.ValueString(),
			OriginFrameworkID: t.OriginFrameworkID.ValueString(),
			ReferenceID:       t.ReferenceID.ValueString(),
		})
	}
	return out
}

func emptyNested() []api_client.CustomComplianceFrameworkSection {
	return []api_client.CustomComplianceFrameworkSection{}
}

func sectionsToAPI(sections []sectionModel) []api_client.CustomComplianceFrameworkSection {
	out := make([]api_client.CustomComplianceFrameworkSection, 0, len(sections))
	for i, s := range sections {
		id := positionalSectionID("", i)
		nested := emptyNested()
		for j, mid := range s.Sections {
			midID := positionalSectionID(id, j)
			leaves := emptyNested()
			for k, leaf := range mid.Sections {
				leafID := positionalSectionID(midID, k)
				leaves = append(leaves, api_client.CustomComplianceFrameworkSection{
					Name:     leaf.Name.ValueString(),
					Tests:    testsToAPI(leafID, leaf.Tests),
					Sections: emptyNested(), // fourth level is rejected by ValidateConfig
				})
			}
			nested = append(nested, api_client.CustomComplianceFrameworkSection{
				Name:     mid.Name.ValueString(),
				Tests:    testsToAPI(midID, mid.Tests),
				Sections: leaves,
			})
		}
		out = append(out, api_client.CustomComplianceFrameworkSection{
			Name:     s.Name.ValueString(),
			Tests:    testsToAPI(id, s.Tests),
			Sections: nested,
		})
	}
	return out
}

func optionalNonEmpty(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func testsFromCatalog(tests []api_client.ComplianceCatalogTest) []testModel {
	if len(tests) == 0 {
		return nil
	}
	out := make([]testModel, len(tests))
	for i, t := range tests {
		out[i] = testModel{
			RuleID:            types.StringValue(t.RuleID),
			RuleIDInFramework: optionalNonEmpty(t.ReferenceID),
			Priority:          optionalNonEmpty(t.Priority),
			ControlUniqueID:   optionalNonEmpty(t.ControlUniqueID),
			OriginFrameworkID: optionalNonEmpty(t.OriginFrameworkID),
			ReferenceID:       optionalNonEmpty(t.ReferenceID),
		}
	}
	return out
}

func sectionsFromCatalog(sections []api_client.ComplianceCatalogSection) []sectionModel {
	if len(sections) == 0 {
		return nil
	}
	out := make([]sectionModel, len(sections))
	for i, s := range sections {
		out[i] = sectionModel{
			Name:  types.StringValue(s.Name),
			Tests: testsFromCatalog(s.Tests),
		}
		if len(s.Sections) == 0 {
			continue
		}
		mids := make([]midSectionModel, len(s.Sections))
		for j, mid := range s.Sections {
			mids[j] = midSectionModel{
				Name:  types.StringValue(mid.Name),
				Tests: testsFromCatalog(mid.Tests),
			}
			if len(mid.Sections) == 0 {
				continue
			}
			leaves := make([]leafSectionModel, len(mid.Sections))
			for k, leaf := range mid.Sections {
				leaves[k] = leafSectionModel{
					Name:  types.StringValue(leaf.Name),
					Tests: testsFromCatalog(leaf.Tests),
				}
			}
			mids[j].Sections = leaves
		}
		out[i].Sections = mids
	}
	return out
}
