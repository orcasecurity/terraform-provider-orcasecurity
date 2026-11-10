package shift_left_cve_exception_list

import (
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func listWith(omitted bool, listDisabled bool, entries ...bool) *api_client.ShiftLeftCveExceptionListRead {
	l := &api_client.ShiftLeftCveExceptionListRead{}
	if !omitted {
		l.Disabled = &listDisabled
	}
	for _, d := range entries {
		l.Vulnerabilities = append(l.Vulnerabilities, api_client.Vulnerability{CVEID: "cve-1", Disabled: d})
	}
	return l
}

func TestReadListDisabled(t *testing.T) {
	cases := []struct {
		name     string
		prior    types.Bool
		instance *api_client.ShiftLeftCveExceptionListRead
		want     bool
	}{
		{"field present is read as-is", types.BoolValue(false), listWith(false, true, false), true},
		{"omitted keeps configured true while every entry is disabled", types.BoolValue(true), listWith(true, false, true, true), true},
		{"omitted shows drift when an entry was re-enabled", types.BoolValue(true), listWith(true, false, true, false), false},
		{"omitted keeps configured false", types.BoolValue(false), listWith(true, false, true), false},
		{"import with every entry disabled", types.BoolNull(), listWith(true, false, true), true},
		{"omitted keeps configured true with no entries", types.BoolValue(true), listWith(true, false), true},
		{"import with no entries", types.BoolNull(), listWith(true, false), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := readListDisabled(c.prior, c.instance); got != c.want {
				t.Fatalf("got %t, want %t", got, c.want)
			}
		})
	}
}

func TestReadVulnerabilityDisabled(t *testing.T) {
	entry := api_client.Vulnerability{CVEID: "cve-1", Disabled: true}
	if readVulnerabilityDisabled(entry, true, types.BoolValue(false)) {
		t.Error("an entry disabled only by the list-level flag must keep its configured false")
	}
	if !readVulnerabilityDisabled(entry, false, types.BoolValue(false)) {
		t.Error("without the list-level flag a disabled entry is drift and must read true")
	}
	if !readVulnerabilityDisabled(entry, true, types.BoolNull()) {
		t.Error("import must read the server value")
	}
}
