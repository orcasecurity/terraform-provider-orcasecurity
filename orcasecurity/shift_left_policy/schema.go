package shift_left_policy

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// allControlsAttr is a section-level toggle that tells the provider to include
// every catalog control for that section, so users don't need a data source or
// to list controls manually.
func allControlsAttr() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"all_controls": schema.BoolAttribute{
			Optional:    true,
			Description: "When true, include every catalog control for this section automatically (no need to list controls or use a data source).",
		},
	}
}

func conditionsBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"fix_available": schema.BoolAttribute{
				Optional: true,
			},
			"from_base_image": schema.BoolAttribute{
				Optional: true,
			},
			"days_from_discovery": schema.Int64Attribute{
				Optional: true,
			},
			"days_from_fix": schema.Int64Attribute{
				Optional: true,
			},
			"has_exploit": schema.BoolAttribute{
				Optional: true,
			},
			"severities_operator": schema.StringAttribute{
				Optional:    true,
				Description: "Severity filter operator (e.g. IN, NOT_IN).",
			},
			"severities_values": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Severity values for the filter (e.g. CRITICAL, HIGH).",
			},
		},
	}
}

func baseControlAttributes(extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:    true,
			Description: "Catalog control ID. Omit to define a custom control identified by its title and conditions.",
		},
		"title": schema.StringAttribute{
			Optional:    true,
			Description: "Control title. Informational for catalog controls (filled from the Orca catalog); required to identify a custom control when no id is set.",
		},
		"priority": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.OneOf("LOW", "MEDIUM", "HIGH", "CRITICAL", "INFO"),
			},
		},
		"disabled": schema.BoolAttribute{
			Required: true,
		},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

func baseControlsListBlock(extra map[string]schema.Attribute) schema.Block {
	return schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: baseControlAttributes(extra),
			Blocks: map[string]schema.Block{
				"conditions": conditionsBlock(),
			},
		},
	}
}

func iacBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: allControlsAttr(),
		Blocks: map[string]schema.Block{
			"controls": baseControlsListBlock(map[string]schema.Attribute{
				"frameworks": schema.ListAttribute{
					ElementType: types.StringType,
					Optional:    true,
				},
				"orca_alert_rule_type": schema.StringAttribute{
					Optional: true,
				},
			}),
		},
	}
}

func sastBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: allControlsAttr(),
		Blocks: map[string]schema.Block{
			"controls": baseControlsListBlock(map[string]schema.Attribute{
				"languages":  schema.ListAttribute{ElementType: types.StringType, Optional: true},
				"owasp":      schema.ListAttribute{ElementType: types.StringType, Optional: true},
				"cwe":        schema.ListAttribute{ElementType: types.StringType, Optional: true},
				"section":    schema.StringAttribute{Optional: true},
				"confidence": schema.StringAttribute{Optional: true},
				"impact":     schema.StringAttribute{Optional: true},
				"likelihood": schema.StringAttribute{Optional: true},
			}),
		},
	}
}

func controlsOnlyBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: allControlsAttr(),
		Blocks: map[string]schema.Block{
			"controls": baseControlsListBlock(nil),
		},
	}
}

func containerScopeBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: allControlsAttr(),
		Blocks: map[string]schema.Block{
			"controls": baseControlsListBlock(map[string]schema.Attribute{
				"origin": schema.StringAttribute{Optional: true},
			}),
		},
	}
}

func containerImageBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"feature_scope": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Enabled feature scopes: vulnerabilities, secret_detection, container_image_best_practices, custom.",
			},
		},
		Blocks: map[string]schema.Block{
			"vulnerabilities":                containerScopeBlock(),
			"secret_detection":               containerScopeBlock(),
			"container_image_best_practices": containerScopeBlock(),
			"custom":                         containerScopeBlock(),
		},
	}
}

func scmPostureBlock() schema.Block {
	return schema.SingleNestedBlock{
		Blocks: map[string]schema.Block{
			"scope": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key": schema.StringAttribute{
							Required: true,
							Description: "Scope key. One of: github_installations, github_repository_installations, " +
								"gitlab_groups, gitlab_projects, azure_organizations, azure_projects.",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"github_installations",
									"github_repository_installations",
									"gitlab_groups",
									"gitlab_projects",
									"azure_organizations",
									"azure_projects",
								),
							},
						},
						"ids": schema.ListAttribute{
							ElementType: types.StringType,
							Required:    true,
						},
					},
				},
			},
			"controls": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Required: true},
						"priority": schema.StringAttribute{Required: true},
						"disabled": schema.BoolAttribute{Required: true},
						"scm":      schema.StringAttribute{Optional: true},
						"entity":   schema.StringAttribute{Optional: true},
						"threat": schema.ListAttribute{
							ElementType: types.StringType,
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func licensesBlock() schema.Block {
	return schema.SingleNestedBlock{
		Attributes: allControlsAttr(),
		Blocks: map[string]schema.Block{
			"controls": baseControlsListBlock(map[string]schema.Attribute{
				"license_id":       schema.StringAttribute{Optional: true},
				"license_category": schema.StringAttribute{Optional: true},
				"is_osi_approved":  schema.BoolAttribute{Optional: true},
				"is_deprecated":    schema.BoolAttribute{Optional: true},
				"is_fsf_libre":     schema.BoolAttribute{Optional: true},
				"url":              schema.StringAttribute{Optional: true},
				"additional_info":  schema.ListAttribute{ElementType: types.StringType, Optional: true},
			}),
		},
	}
}

func resourceSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "AppSec policy ID.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"type": schema.StringAttribute{
			Required:    true,
			Description: "Policy type.",
			Validators: []validator.String{
				stringvalidator.OneOf(policyTypes...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "Policy name.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Description: "Policy description.",
		},
		"disabled": schema.BoolAttribute{
			Required:    true,
			Description: "Whether the policy is disabled.",
		},
		"warn_mode": schema.BoolAttribute{
			Required:    true,
			Description: "When true, policy violations produce warnings instead of failures.",
		},
		"priority_failure_threshold": schema.StringAttribute{
			Required:    true,
			Description: "Minimum control priority that causes scan failure.",
			Validators: []validator.String{
				stringvalidator.OneOf("LOW", "MEDIUM", "HIGH", "CRITICAL"),
			},
		},
		"projects_ids": schema.SetAttribute{
			ElementType: types.StringType,
			Optional:    true,
			Computed:    true,
			Description: "Project IDs to attach this policy to. Reflects the API on read; omit to leave the current attachment unchanged, or set to `[]` to detach from all projects.",
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.UseStateForUnknown(),
			},
		},
		"builtin": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether this is an Orca built-in policy. Built-in policies cannot be renamed or deleted via Terraform; other attributes remain updatable.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func resourceSchemaBlocks() map[string]schema.Block {
	return map[string]schema.Block{
		"iac":                          iacBlock(),
		"sast":                         sastBlock(),
		"file_system_vulnerabilities":  controlsOnlyBlock(),
		"file_system_secret_detection": controlsOnlyBlock(),
		"container_image":              containerImageBlock(),
		"scm_posture":                  scmPostureBlock(),
		"licenses":                     licensesBlock(),
	}
}
