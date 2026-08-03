package shift_left_common

import (
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Terraform core before 1.4 does not carry a prior state value forward for a Computed attribute on
// its own. Without an explicit carry-forward modifier every unset Computed attribute replans as
// "known after apply", so an apply against those versions never produces an empty follow-up plan.
// These constructors make the modifier part of declaring the attribute, so a new SCM attribute
// cannot quietly reintroduce that perpetual diff. TestShiftLeftScmSchemasSettle pins the invariant.

func ComputedString(description string) rschema.StringAttribute {
	return rschema.StringAttribute{
		Computed:      true,
		Description:   description,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func OptionalComputedString(description string, validators ...validator.String) rschema.StringAttribute {
	attr := ComputedString(description)
	attr.Optional = true
	attr.Validators = validators
	return attr
}

func ComputedBool(description string) rschema.BoolAttribute {
	return rschema.BoolAttribute{
		Computed:      true,
		Description:   description,
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func OptionalComputedBool(description string) rschema.BoolAttribute {
	attr := ComputedBool(description)
	attr.Optional = true
	return attr
}

func ComputedInt64(description string) rschema.Int64Attribute {
	return rschema.Int64Attribute{
		Computed:      true,
		Description:   description,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}
