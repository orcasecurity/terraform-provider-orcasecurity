package api_client_test

import (
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"testing"
)

func TestSplitComplianceSection(t *testing.T) {
	tests := []struct {
		name                                            string
		section                                         string
		wantCategory, wantSubCategory, wantSubSubCatego string
	}{
		{"three levels", "Identify/Risk Assessment/Vulnerabilities are identified", "Identify", "Risk Assessment", "Vulnerabilities are identified"},
		{"two levels", "Identify/Risk Assessment", "Identify", "Risk Assessment", ""},
		{"single level", "section_2", "section_2", "", ""},
		{"deeper than three keeps remainder in sub_sub", "A/B/C/D", "A", "B", "C/D"},
		{"empty", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, sc, ssc := api_client.SplitComplianceSection(tt.section)
			if c != tt.wantCategory || sc != tt.wantSubCategory || ssc != tt.wantSubSubCatego {
				t.Fatalf("SplitComplianceSection(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.section, c, sc, ssc, tt.wantCategory, tt.wantSubCategory, tt.wantSubSubCatego)
			}
		})
	}
}

func TestComplianceSectionRoundTrip(t *testing.T) {
	sections := []string{
		"Identify/Risk Assessment/Vulnerabilities are identified",
		"Identify/Risk Assessment",
		"section_2",
		"A/B/C/D",
		"",
	}
	for _, section := range sections {
		c, sc, ssc := api_client.SplitComplianceSection(section)
		got := api_client.JoinComplianceSection(c, sc, ssc)
		if got != section {
			t.Fatalf("round trip of %q = %q", section, got)
		}
	}
}
