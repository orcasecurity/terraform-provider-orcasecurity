package alert_common

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Attributes the discovery and sonar custom alert resources share verbatim. The
// API payload is the same for both, so the schema has to be too.

// ComplianceFrameworksAttribute is Optional+Computed on purpose: the links are
// also written by the Orca UI and by the custom compliance framework resource,
// and an Optional-only attribute makes Terraform delete what the config does not
// declare (WASP-1672).
func ComplianceFrameworksAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: "The custom compliance framework(s) that this alert relates to. In the context of a compliance framework, alerts correspond to controls. Omit the attribute to leave the existing links untouched - they may be owned by the Orca UI or by a custom compliance framework resource. Set it to `[]` to detach the alert from every framework.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Required:    true,
					Description: "Custom framework name.",
				},
				"section": schema.StringAttribute{
					Required:    true,
					Description: "Custom framework section. For nested sections, join the levels with `/` (e.g. `Identify/Risk Assessment/Vulnerabilities in assets are identified`); up to three levels are supported.",
				},
				"priority": schema.StringAttribute{
					Required:    true,
					Description: "Custom framework control priority. Valid values are `high`, `medium`, and `low`.",
				},
			},
		},
	}
}

func RemediationTextAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "A container for the remediation instructions that will appear on the 'Remediation' tab for the alert.",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"enable": schema.BoolAttribute{
				Description: "Whether or not all users are able to see the remediation instructions for this alert. To enable all users to see them, set this to `true`.",
				Optional:    true,
			},
			"text": schema.StringAttribute{
				Description: "Remediation description.",
				Required:    true,
			},
		},
	}
}

// MergeAttributes merges a resource's own attributes onto the ones every custom
// alert resource declares identically.
func MergeAttributes(own map[string]schema.Attribute) map[string]schema.Attribute {
	attributes := CommonAttributes()
	for name, attribute := range own {
		attributes[name] = attribute
	}
	return attributes
}

// CommonAttributes are the attributes every custom alert resource declares
// identically. Callers merge them into their own attribute map and add the ones
// that differ (rule_type, orca_score, context_score, and the alert's own rule).
func CommonAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Description: "Custom alert ID.",
		},
		"organization_id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
			Description: "Orca organization ID.",
		},
		"name": schema.StringAttribute{
			Description: "Custom alert name.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
			},
		},
		"description": schema.StringAttribute{
			Description: "Custom alert description.",
			Optional:    true,
		},
		"category": schema.StringAttribute{
			Description: "Alert category. Valid values are `Access control`, `Authentication`, `Best practices`, `Data at risk`, `Data protection`, `IAM misconfigurations`, `Lateral movement`, `Logging and monitoring`, `Malicious activity`, `Malware`, `Neglected assets`, `Network misconfigurations`, `Source code vulnerabilities`, `Suspicious activity`, `System integrity`, `Vendor services misconfigurations`, `Vulnerabilities`, and `Workload misconfigurations`.",
			Required:    true,
		},
	}
}
