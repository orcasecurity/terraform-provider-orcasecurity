package custom_discovery_alert

import (
	"context"
	"testing"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func complianceFrameworksAttribute(t *testing.T) schema.ListNestedAttribute {
	t.Helper()
	r := &customDiscoveryAlertResource{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	attribute, ok := schemaResp.Schema.Attributes["compliance_frameworks"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("compliance_frameworks must be a ListNestedAttribute")
	}
	return attribute
}

// WASP-1672: without Computed, a config that never declares the attribute plans
// the remote link away and the apply clears it on the backend.
func TestComplianceFrameworksIsOptionalComputed(t *testing.T) {
	attribute := complianceFrameworksAttribute(t)
	if !attribute.Optional {
		t.Error("compliance_frameworks must stay Optional")
	}
	if !attribute.Computed {
		t.Error("compliance_frameworks must be Computed so a silent config keeps the remote links")
	}
	var hasUseStateForUnknown bool
	for _, modifier := range attribute.PlanModifiers {
		if modifier == listplanmodifier.UseStateForUnknown() {
			hasUseStateForUnknown = true
		}
	}
	if !hasUseStateForUnknown {
		t.Error("compliance_frameworks needs UseStateForUnknown so updates do not replan the links")
	}
}

func TestFrameworksToListJoinsSectionLevels(t *testing.T) {
	list, diags := frameworksToList(context.Background(), []api_client.CustomDiscoveryAlertComplianceFramework{{
		Name:           "Bayer GCP CSR",
		Category:       "Identify",
		SubCategory:    "Risk Assessment",
		SubSubCategory: "Vulnerabilities in assets are identified",
		Priority:       "medium",
	}})
	if diags.HasError() {
		t.Fatal(diags)
	}
	frameworks, diags := generateRequestFrameworks(context.Background(), list)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(frameworks) != 1 {
		t.Fatalf("expected 1 framework, got %d", len(frameworks))
	}
	got := frameworks[0]
	if got.Category != "Identify" || got.SubCategory != "Risk Assessment" ||
		got.SubSubCategory != "Vulnerabilities in assets are identified" {
		t.Fatalf("section levels did not survive the round trip: %+v", got)
	}
}

// The request must stay empty for a silent config: an unknown plan value is what
// Terraform hands the provider on create, a null one on update.
func TestGenerateRequestFrameworksSkipsSilentConfig(t *testing.T) {
	for name, list := range map[string]types.List{
		"null":    types.ListNull(complianceFrameworksAttribute(t).NestedObject.Type()),
		"unknown": types.ListUnknown(complianceFrameworksAttribute(t).NestedObject.Type()),
	} {
		t.Run(name, func(t *testing.T) {
			frameworks, diags := generateRequestFrameworks(context.Background(), list)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if frameworks != nil {
				t.Fatalf("expected no frameworks in the request, got %v", frameworks)
			}
		})
	}
}
