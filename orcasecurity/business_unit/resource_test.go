package business_unit_test

import (
	"fmt"
	"regexp"
	"terraform-provider-orcasecurity/orcasecurity"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

const (
	ResourceType             = "orcasecurity_business_unit"
	Resource                 = "terraformTestResource"
	OrcaObject1              = "terraformTestResourceInOrcaAws"
	OrcaObject2              = "terraformTestResourceInOrcaAzure"
	OrcaObject3              = "terraformTestResourceInOrcaShiftLeftProjects"
	OrcaObject4              = "terraformTestResourceDeprecatedCloudAccountIds"
	shiftLeftProjectIDsAttr0 = "shiftleft_filter_data.shiftleft_project_ids.0"
	shiftLeftProjectIDsAttr1 = "shiftleft_filter_data.shiftleft_project_ids.1"
)

func TestAccBusinessUnitResourceInvalidShiftLeftProjectIDs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    shiftleft_filter_data = {
        shiftleft_project_ids = ["test"]
    }
}`, ResourceType, Resource, OrcaObject3),
				ExpectError: regexp.MustCompile("Invalid UUID|must be a valid UUID"),
			},
		},
	})
}

func TestAccBusinessUnitResource_CloudProvider(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    filter_data = {
        cloud_providers = ["aws"]
    }
}`, ResourceType, Resource, OrcaObject1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject1),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_providers.0", "aws"),
				),
			},
			// import
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    filter_data = {
        cloud_providers = ["azure"]
    }
}`, ResourceType, Resource, OrcaObject2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject2),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_providers.0", "azure"),
				),
			},
		},
	})
}

func TestAccBusinessUnitResource_ShiftLeft(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "bu_project" {
    name             = "BU Test Project"
    description      = "Project for business unit test"
    key              = "bu-test-project"
    default_policies = true
}

resource "%s" "%s" {
    name = "%s"
    shiftleft_filter_data = {
        shiftleft_project_ids = [orcasecurity_shift_left_project.bu_project.id]
    }
}`, ResourceType, Resource, OrcaObject3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fmt.Sprintf("%s.%s", ResourceType, Resource), shiftLeftProjectIDsAttr0),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "bu_project" {
    name             = "BU Test Project"
    description      = "Project for business unit test"
    key              = "bu-test-project"
    default_policies = true
}

resource "orcasecurity_shift_left_project" "bu_project_2" {
    name             = "BU Test Project 2"
    description      = "Second project for BU update test"
    key              = "bu-test-project-2"
    default_policies = true
}

resource "%s" "%s" {
    name = "%s"
    shiftleft_filter_data = {
        shiftleft_project_ids = [
            orcasecurity_shift_left_project.bu_project.id,
            orcasecurity_shift_left_project.bu_project_2.id
        ]
    }
}`, ResourceType, Resource, OrcaObject3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fmt.Sprintf("%s.%s", ResourceType, Resource), shiftLeftProjectIDsAttr0),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("%s.%s", ResourceType, Resource), shiftLeftProjectIDsAttr1),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "name", OrcaObject3),
				),
			},
		},
	})
}

func TestAccBusinessUnitResource_CloudAndShiftLeft(t *testing.T) {
	// Test env must have this cloud account (cloud_vendor_id) in Orca
	cloudVendorID := "550e8400-e29b-41d4-a716-446655440000"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "orcasecurity_shift_left_project" "bu_project" {
    name             = "BU Cloud and SL Test Project"
    description      = "Project for business unit cloud+shiftleft test"
    key              = "bu-cloud-sl-project"
    default_policies = true
}

resource "%s" "%s" {
    name = "%s"
    filter_data = {
        cloud_vendor_id = ["%s"]
    }
    shiftleft_filter_data = {
        shiftleft_project_ids = [orcasecurity_shift_left_project.bu_project.id]
    }
}`, ResourceType, Resource, OrcaObject3, cloudVendorID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fmt.Sprintf("%s.%s", ResourceType, Resource), shiftLeftProjectIDsAttr0),
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_vendor_id.0", cloudVendorID),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			{
				ResourceName:      fmt.Sprintf("%s.%s", ResourceType, Resource),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBusinessUnitResourceCloudAccountIdsDeprecatedNoDrift(t *testing.T) {
	// Verifies deprecated cloud_account_ids doesn't cause perpetual drift
	cloudVendorID := "550e8400-e29b-41d4-a716-446655440000"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: orcasecurity.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orcasecurity.TestProviderConfig + fmt.Sprintf(`
resource "%s" "%s" {
    name = "%s"
    filter_data = {
        cloud_account_ids = ["%s"]
    }
}`, ResourceType, Resource, OrcaObject4, cloudVendorID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("%s.%s", ResourceType, Resource), "filter_data.cloud_account_ids.0", cloudVendorID),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
