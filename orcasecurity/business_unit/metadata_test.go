package business_unit

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func TestApplyMetadataToRequest_Populated(t *testing.T) {
	globalFilter := true
	plan := &businessUnitResourceModel{
		GlobalFilter:        types.BoolValue(globalFilter),
		BusinessCriticality: types.StringValue("high"),
		OwnerTeam:           types.StringValue("platform"),
		Application:         types.StringValue("billing"),
		ContactEmails:       []types.String{types.StringValue("a@x.com"), types.StringValue("b@x.com")},
		DeploymentStages:    []types.String{types.StringValue("dev"), types.StringValue("prod")},
	}

	req := api_client.BusinessUnit{Name: "test"}
	applyMetadataToRequest(&req, plan)

	if req.GlobalFilter == nil || *req.GlobalFilter != true {
		t.Errorf("expected GlobalFilter true, got %v", req.GlobalFilter)
	}
	if req.BusinessCriticality != "high" {
		t.Errorf("expected high, got %q", req.BusinessCriticality)
	}
	if req.OwnerTeam != "platform" {
		t.Errorf("expected platform, got %q", req.OwnerTeam)
	}
	if req.Application != "billing" {
		t.Errorf("expected billing, got %q", req.Application)
	}
	if !reflect.DeepEqual(req.ContactEmails, []string{"a@x.com", "b@x.com"}) {
		t.Errorf("contact emails mismatch: %v", req.ContactEmails)
	}
	if !reflect.DeepEqual(req.DeploymentStages, []string{"dev", "prod"}) {
		t.Errorf("deployment stages mismatch: %v", req.DeploymentStages)
	}
}

func TestApplyMetadataToRequest_NullPlanProducesEmptyValues(t *testing.T) {
	plan := &businessUnitResourceModel{
		GlobalFilter:        types.BoolNull(),
		BusinessCriticality: types.StringNull(),
		OwnerTeam:           types.StringNull(),
		Application:         types.StringNull(),
		ContactEmails:       nil,
		DeploymentStages:    nil,
	}

	req := api_client.BusinessUnit{Name: "test"}
	applyMetadataToRequest(&req, plan)

	if req.GlobalFilter != nil {
		t.Errorf("expected nil GlobalFilter, got %v", req.GlobalFilter)
	}
	if req.BusinessCriticality != "" {
		t.Errorf("expected empty BusinessCriticality, got %q", req.BusinessCriticality)
	}
	if req.OwnerTeam != "" {
		t.Errorf("expected empty OwnerTeam, got %q", req.OwnerTeam)
	}
	if req.Application != "" {
		t.Errorf("expected empty Application, got %q", req.Application)
	}
	if len(req.ContactEmails) != 0 {
		t.Errorf("expected empty ContactEmails, got %v", req.ContactEmails)
	}
	if len(req.DeploymentStages) != 0 {
		t.Errorf("expected empty DeploymentStages, got %v", req.DeploymentStages)
	}
}

func TestBusinessUnit_NullPlanSerializesEmptyMetadata(t *testing.T) {
	plan := &businessUnitResourceModel{
		Name:                types.StringValue("test"),
		GlobalFilter:        types.BoolNull(),
		BusinessCriticality: types.StringNull(),
		OwnerTeam:           types.StringNull(),
		Application:         types.StringNull(),
		ContactEmails:       nil,
		DeploymentStages:    nil,
	}

	req := api_client.BusinessUnit{Name: plan.Name.ValueString()}
	applyMetadataToRequest(&req, plan)

	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)

	for _, want := range []string{
		`"business_criticality":""`,
		`"owner_team":""`,
		`"application":""`,
		`"contact_emails":[]`,
		`"deployment_stages":[]`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected JSON to contain %s, got %s", want, got)
		}
	}
	if strings.Contains(got, `"global_filter"`) {
		t.Errorf("expected global_filter omitted when null, got %s", got)
	}
}

func TestSetMetadataInState_EmptyAPIValuesYieldNull(t *testing.T) {
	state := &businessUnitResourceModel{}
	instance := &api_client.BusinessUnit{}

	setMetadataInState(state, instance)

	if !state.GlobalFilter.IsNull() {
		t.Errorf("expected GlobalFilter null, got %v", state.GlobalFilter)
	}
	if !state.BusinessCriticality.IsNull() {
		t.Errorf("expected BusinessCriticality null, got %v", state.BusinessCriticality)
	}
	if !state.OwnerTeam.IsNull() {
		t.Errorf("expected OwnerTeam null, got %v", state.OwnerTeam)
	}
	if !state.Application.IsNull() {
		t.Errorf("expected Application null, got %v", state.Application)
	}
	if state.ContactEmails != nil {
		t.Errorf("expected ContactEmails nil, got %v", state.ContactEmails)
	}
	if state.DeploymentStages != nil {
		t.Errorf("expected DeploymentStages nil, got %v", state.DeploymentStages)
	}
}

func TestSetMetadataInState_PopulatedAPIValues(t *testing.T) {
	gf := true
	state := &businessUnitResourceModel{}
	instance := &api_client.BusinessUnit{
		GlobalFilter:        &gf,
		BusinessCriticality: "critical",
		OwnerTeam:           "sec",
		Application:         "infra",
		ContactEmails:       []string{"alert@x.com"},
		DeploymentStages:    []string{"staging"},
	}

	setMetadataInState(state, instance)

	if !state.GlobalFilter.Equal(types.BoolValue(true)) {
		t.Errorf("expected GlobalFilter true, got %v", state.GlobalFilter)
	}
	if state.BusinessCriticality.ValueString() != "critical" {
		t.Errorf("expected critical, got %s", state.BusinessCriticality.ValueString())
	}
	if state.OwnerTeam.ValueString() != "sec" {
		t.Errorf("expected sec, got %s", state.OwnerTeam.ValueString())
	}
	if state.Application.ValueString() != "infra" {
		t.Errorf("expected infra, got %s", state.Application.ValueString())
	}
	if len(state.ContactEmails) != 1 || state.ContactEmails[0].ValueString() != "alert@x.com" {
		t.Errorf("contact emails mismatch: %v", state.ContactEmails)
	}
	if len(state.DeploymentStages) != 1 || state.DeploymentStages[0].ValueString() != "staging" {
		t.Errorf("deployment stages mismatch: %v", state.DeploymentStages)
	}
}
