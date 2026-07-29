package shift_left_policy

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
	"terraform-provider-orcasecurity/orcasecurity/tfconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// blockShape describes a flat (non-scoped) policy type whose block B carries an
// all_controls flag and a flat control list. flatHandler builds the handler from
// it so the five flat types collapse to one line each.
type blockShape[B any] struct {
	get         func(*shiftLeftPolicyResourceModel) *B
	set         func(*shiftLeftPolicyResourceModel, *B)
	allControls func(*B) types.Bool
	toMaps      func(*B) []map[string]interface{}
	build       func([]map[string]interface{}) *B
	merge       func(dst, src *B)
}

func flatHandler[B any](catalog string, s blockShape[B]) policyTypeHandler {
	return policyTypeHandler{
		catalogType: catalog,
		present:     func(m *shiftLeftPolicyResourceModel) bool { return s.get(m) != nil },
		allControlsScopes: singleScopeAll(func(m *shiftLeftPolicyResourceModel) bool {
			b := s.get(m)
			return b != nil && tfconv.BoolIsTrue(s.allControls(b))
		}),
		buildWrite: controlsWrite(func(m *shiftLeftPolicyResourceModel) []map[string]interface{} {
			return s.toMaps(s.get(m))
		}),
		applyRead: func(m *shiftLeftPolicyResourceModel, _ *api_client.ShiftLeftPolicy, _ map[string]interface{}, controls []map[string]interface{}) {
			s.set(m, s.build(controls))
		},
		mergePlan: func(state, plan *shiftLeftPolicyResourceModel) {
			s.merge(s.get(state), s.get(plan))
		},
	}
}

type policyTypeHandler struct {
	catalogType       string
	present           func(m *shiftLeftPolicyResourceModel) bool
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
		present:     func(m *shiftLeftPolicyResourceModel) bool { return get(m) != nil },
		allControlsScopes: func(m *shiftLeftPolicyResourceModel) []string {
			if b := get(m); b != nil && tfconv.BoolIsTrue(b.AllControls) {
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
	"iac": flatHandler("iac", blockShape[iacBlockModel]{
		get:         func(m *shiftLeftPolicyResourceModel) *iacBlockModel { return m.Iac },
		set:         func(m *shiftLeftPolicyResourceModel, b *iacBlockModel) { m.Iac = b },
		allControls: func(b *iacBlockModel) types.Bool { return b.AllControls },
		toMaps:      iacControlsToMaps,
		build:       buildIacBlock,
		merge:       mergeIacBlockFromPlan,
	}),
	"sast": flatHandler("sast", blockShape[sastBlockModel]{
		get:         func(m *shiftLeftPolicyResourceModel) *sastBlockModel { return m.Sast },
		set:         func(m *shiftLeftPolicyResourceModel, b *sastBlockModel) { m.Sast = b },
		allControls: func(b *sastBlockModel) types.Bool { return b.AllControls },
		toMaps:      sastControlsToMaps,
		build:       buildSastBlock,
		merge:       mergeSastBlockFromPlan,
	}),
	// Legacy aggregate file_system type: flat controls (no feature_scope), unlike
	// the scoped file_system_* sub-types. Kept for backward compatibility.
	"file_system": flatHandler("file_system", blockShape[controlsBlockModel]{
		get:         func(m *shiftLeftPolicyResourceModel) *controlsBlockModel { return m.FileSystem },
		set:         func(m *shiftLeftPolicyResourceModel, b *controlsBlockModel) { m.FileSystem = b },
		allControls: func(b *controlsBlockModel) types.Bool { return b.AllControls },
		toMaps:      controlsBlockToMaps,
		build:       buildControlsBlock,
		merge:       mergeControlsBlockFromPlan,
	}),
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
		present:     func(m *shiftLeftPolicyResourceModel) bool { return m.ContainerImage != nil },
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
		present:     func(m *shiftLeftPolicyResourceModel) bool { return m.ScmPosture != nil },
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
	"licenses": flatHandler("licenses", blockShape[licensesBlockModel]{
		get:         func(m *shiftLeftPolicyResourceModel) *licensesBlockModel { return m.Licenses },
		set:         func(m *shiftLeftPolicyResourceModel, b *licensesBlockModel) { m.Licenses = b },
		allControls: func(b *licensesBlockModel) types.Bool { return b.AllControls },
		toMaps:      func(b *licensesBlockModel) []map[string]interface{} { return licenseControlsToMaps(b.Controls) },
		build:       buildLicensesBlock,
		merge:       mergeLicensesBlockFromPlan,
	}),
	// Legacy sca type: shares the licenses block shape (superseded by licenses).
	// Kept for backward compatibility.
	"sca": flatHandler("sca", blockShape[licensesBlockModel]{
		get:         func(m *shiftLeftPolicyResourceModel) *licensesBlockModel { return m.Sca },
		set:         func(m *shiftLeftPolicyResourceModel, b *licensesBlockModel) { m.Sca = b },
		allControls: func(b *licensesBlockModel) types.Bool { return b.AllControls },
		toMaps:      func(b *licensesBlockModel) []map[string]interface{} { return licenseControlsToMaps(b.Controls) },
		build:       buildLicensesBlock,
		merge:       mergeLicensesBlockFromPlan,
	}),
	// malicious_packages: no controls, no catalog; policy_data omitted (server defaults it to {}).
	"malicious_packages": {},
}

func containerAllControlsScopes(block *containerImageBlockModel) []string {
	if block == nil {
		return nil
	}
	var keys []string
	scopes := []struct {
		key   string
		block *containerScopeBlockModel
	}{
		{"vulnerabilities", block.Vulnerabilities},
		{"secret_detection", block.SecretDetection},
		{"container_image_best_practices", block.ContainerImageBestPractices},
		{"custom", block.Custom},
	}
	for _, s := range scopes {
		if s.block != nil && tfconv.BoolIsTrue(s.block.AllControls) {
			keys = append(keys, s.key)
		}
	}
	return keys
}

func validateTypeBlock(policyType string, model *shiftLeftPolicyResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	h, ok := policyTypeHandlers[policyType]
	if !ok {
		diags.AddError("Unsupported policy type", fmt.Sprintf("Unknown policy type %q.", policyType))
		return diags
	}
	if h.present != nil && !h.present(model) {
		diags.AddError("Missing type configuration block", fmt.Sprintf("Policy type %q requires the %q block to be set.", policyType, policyType))
	}
	return diags
}

func buildControlsAndData(model *shiftLeftPolicyResourceModel, policy *api_client.ShiftLeftPolicy) ([]map[string]interface{}, map[string]interface{}, diag.Diagnostics) {
	policyData := map[string]interface{}{}
	buildWrite := policyTypeHandlers[model.Type.ValueString()].buildWrite
	if buildWrite == nil {
		return nil, policyData, nil
	}
	controls, diags := buildWrite(model, policy, policyData)
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
