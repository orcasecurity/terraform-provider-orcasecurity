package shift_left_policy

import (
	"fmt"
	"reflect"

	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type policyTypeHandler struct {
	catalogType       string
	block             func(m *shiftLeftPolicyResourceModel) any
	allControlsScopes func(m *shiftLeftPolicyResourceModel) []string
	buildWrite        func(m *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy, policyData map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics)
	applyRead         func(m *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{})
	mergePlan         func(state, plan *shiftLeftPolicyResourceModel)
}

func controlsWrite(toMaps func(m *shiftLeftPolicyResourceModel) []map[string]interface{}) func(*shiftLeftPolicyResourceModel, *api_client.ShiftLeftPolicy, map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
	return func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, policyData map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
		controls := toMaps(m)
		policyData["controls"] = controls
		return controls, nil
	}
}

func singleScopeAll(requested func(m *shiftLeftPolicyResourceModel) bool) func(m *shiftLeftPolicyResourceModel) []string {
	return func(m *shiftLeftPolicyResourceModel) []string {
		if requested(m) {
			return []string{""}
		}
		return nil
	}
}

// file_system_* API requires scoped policy_data; flat controls rejected (400).
func fsScopedHandler(
	scope string,
	get func(m *shiftLeftPolicyResourceModel) *controlsBlockModel,
	set func(m *shiftLeftPolicyResourceModel, b *controlsBlockModel),
) policyTypeHandler {
	return policyTypeHandler{
		catalogType: "file_system",
		block:       func(m *shiftLeftPolicyResourceModel) any { return get(m) },
		allControlsScopes: func(m *shiftLeftPolicyResourceModel) []string {
			if b := get(m); b != nil && boolIsTrue(b.AllControls) {
				return []string{scope}
			}
			return nil
		},
		buildWrite: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, policyData map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
			controls := controlsBlockToMaps(get(m))
			policyData["feature_scope"] = []string{scope}
			policyData[scope] = scopeControlsWrapper(controls)
			return controls, nil
		},
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{}) {
			if scoped := rawScopeControls(policyData, scope); scoped != nil {
				controls = scoped
			}
			set(m, buildControlsBlock(controls))
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeControlsBlockFromPlan(get(state), get(plan))
		},
	}
}

var policyTypeHandlers = map[string]policyTypeHandler{
	"iac": {
		catalogType: "iac",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.Iac },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			return m.Iac != nil && boolIsTrue(m.Iac.AllControls)
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return iacControlsToMaps(m.Iac)
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.Iac = buildIacBlock(controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) { mergeIacBlockFromPlan(state.Iac, plan.Iac) },
	},
	"sast": {
		catalogType: "sast",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.Sast },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			return m.Sast != nil && boolIsTrue(m.Sast.AllControls)
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return sastControlsToMaps(m.Sast)
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.Sast = buildSastBlock(controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) { mergeSastBlockFromPlan(state.Sast, plan.Sast) },
	},
	// Legacy aggregate file_system type: flat controls (no feature_scope), unlike
	// the scoped file_system_* sub-types. Kept for backward compatibility.
	"file_system": {
		catalogType: "file_system",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.FileSystem },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			return m.FileSystem != nil && boolIsTrue(m.FileSystem.AllControls)
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return controlsBlockToMaps(m.FileSystem)
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.FileSystem = buildControlsBlock(controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeControlsBlockFromPlan(state.FileSystem, plan.FileSystem)
		},
	},
	"file_system_vulnerabilities": fsScopedHandler("vulnerabilities",
		func(m *shiftLeftPolicyResourceModel) *controlsBlockModel { return m.FileSystemVulnerabilities },
		func(m *shiftLeftPolicyResourceModel, b *controlsBlockModel) { m.FileSystemVulnerabilities = b },
	),
	"file_system_secret_detection": fsScopedHandler("secret_detection",
		func(m *shiftLeftPolicyResourceModel) *controlsBlockModel { return m.FileSystemSecretDetection },
		func(m *shiftLeftPolicyResourceModel, b *controlsBlockModel) { m.FileSystemSecretDetection = b },
	),
	"container_image": {
		catalogType: "container_image",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.ContainerImage },
		allControlsScopes: func(m *shiftLeftPolicyResourceModel) []string {
			return containerAllControlsScopes(m.ContainerImage)
		},
		buildWrite: func(m *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy, policyData map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
			return buildContainerImageData(m.ContainerImage, policy, policyData), nil
		},
		applyRead: func(m *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{}) {
			m.ContainerImage = buildContainerImageBlock(apiPolicy, policyData, controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeContainerImageFromPlan(state.ContainerImage, plan.ContainerImage)
		},
	},
	"scm_posture": {
		catalogType: "scm_posture",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.ScmPosture },
		buildWrite: func(m *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy, policyData map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
			// Built-in scm_posture policies are org-global: the API requires
			// them to have no scope (only controls/disabled are updatable), so
			// scope is omitted from the write instead of being required.
			if m.Builtin.ValueBool() {
				var diags diag.Diagnostics
				if len(m.ScmPosture.Scope) > 0 {
					diags.AddError(
						"Invalid scope for built-in policy",
						"Built-in scm_posture policies are global and cannot have a scope.",
					)
					return nil, diags
				}
				scmControls := scmControlsToMaps(m.ScmPosture.Controls)
				policyData["controls"] = scmControls
				return scmControls, diags
			}
			scopeRaw, scmControls, diags := buildScmScope(m.ScmPosture)
			if diags.HasError() {
				return nil, diags
			}
			policy.Scope = scopeRaw
			policyData["controls"] = scmControls
			return scmControls, diags
		},
		applyRead: func(m *shiftLeftPolicyResourceModel, apiPolicy *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.ScmPosture = buildScmPostureBlock(apiPolicy, controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeScmPostureBlockFromPlan(state.ScmPosture, plan.ScmPosture)
		},
	},
	"licenses": {
		catalogType: "licenses",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.Licenses },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			return m.Licenses != nil && boolIsTrue(m.Licenses.AllControls)
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return licenseControlsToMaps(m.Licenses.Controls)
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.Licenses = buildLicensesBlock(controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeLicensesBlockFromPlan(state.Licenses, plan.Licenses)
		},
	},
	// Legacy sca type: shares the licenses block shape (superseded by licenses).
	// Kept for backward compatibility.
	"sca": {
		catalogType: "sca",
		block:       func(m *shiftLeftPolicyResourceModel) any { return m.Sca },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			return m.Sca != nil && boolIsTrue(m.Sca.AllControls)
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return licenseControlsToMaps(m.Sca.Controls)
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			m.Sca = buildLicensesBlock(controls)
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			mergeLicensesBlockFromPlan(state.Sca, plan.Sca)
		},
	},
	// malicious_packages: no controls, no catalog, policy_data always {}.
	"malicious_packages": {
		buildWrite: func(*shiftLeftPolicyResourceModel, *api_client.ShiftLeftPolicy, map[string]interface{}) ([]map[string]interface{}, diag.Diagnostics) {
			return nil, nil
		},
	},
}

// Typed-nil block pointers are not == nil when wrapped in any.
func blockIsUnset(v any) bool {
	return v == nil || (reflect.ValueOf(v).Kind() == reflect.Ptr && reflect.ValueOf(v).IsNil())
}

func validateTypeBlock(policyType string, model *shiftLeftPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	h, ok := policyTypeHandlers[policyType]
	if !ok {
		diags.AddError("Unsupported policy type", fmt.Sprintf("Unknown policy type %q.", policyType))
		return diags
	}
	if h.block != nil && blockIsUnset(h.block(model)) {
		diags.AddError("Missing type configuration block", fmt.Sprintf("Policy type %q requires the %q block to be set.", policyType, policyType))
	}
	return diags
}

func buildControlsAndData(model *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy) ([]map[string]interface{}, map[string]interface{}, diag.Diagnostics) {
	policyData := map[string]interface{}{}
	controls, diags := policyTypeHandlers[model.Type.ValueString()].buildWrite(model, policy, policyData)
	if diags.HasError() {
		return nil, nil, diags
	}
	return controls, policyData, diags
}

func allControlsScopeKeys(model *shiftLeftPolicyResourceModel) []string {
	h, ok := policyTypeHandlers[model.Type.ValueString()]
	if !ok || h.allControlsScopes == nil {
		return nil
	}
	return h.allControlsScopes(model)
}

func applyTypeBlockToState(model *shiftLeftPolicyResourceModel, policyType string, apiPolicy *api_client.ShiftLeftPolicy, policyData map[string]interface{}, controls []map[string]interface{}) {
	if h, ok := policyTypeHandlers[policyType]; ok && h.applyRead != nil {
		h.applyRead(model, apiPolicy, policyData, controls)
	}
}

func mergeStateFromPlan(state, plan *shiftLeftPolicyResourceModel) {
	if h, ok := policyTypeHandlers[plan.Type.ValueString()]; ok && h.mergePlan != nil {
		h.mergePlan(state, plan)
	}
}
