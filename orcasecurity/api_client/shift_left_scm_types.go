package api_client

type ScmPolicyRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Builtin bool   `json:"builtin"`
}

type ScmProjectRef struct {
	ID string `json:"id"`
}

func PolicyRefIDs(refs []ScmPolicyRef) []string {
	if len(refs) == 0 {
		return nil
	}
	ids := make([]string, 0, len(refs))
	for _, r := range refs {
		ids = append(ids, r.ID)
	}
	return ids
}

func ProjectRefID(p *ScmProjectRef) string {
	if p == nil {
		return ""
	}
	return p.ID
}

// PUT sends project_id XOR policies. installation_mode has no server default,
// so the key is always sent (no omitempty); an absent mode is a 400.
type ScmInstallationUpdate struct {
	InstallationMode string                  `json:"installation_mode"`
	DefaultPolicies  bool                    `json:"default_policies"`
	Policies         []string                `json:"policies"`
	ProjectID        string                  `json:"project_id,omitempty"`
	ConfigSettings   ShiftLeftConfigSettings `json:"configuration_settings"`
}

type ScmUnitCommonFields struct {
	InstallationMode  string                  `json:"installation_mode,omitempty"`
	DefaultPolicies   bool                    `json:"default_policies"`
	Policies          []ScmPolicyRef          `json:"policies,omitempty"`
	Project           *ScmProjectRef          `json:"project,omitempty"`
	IntegrationStatus string                  `json:"integration_status,omitempty"`
	ConfigSettings    ShiftLeftConfigSettings `json:"configuration_settings"`

	ScanAllState                string `json:"scan_all_state,omitempty"`
	IntegratedRepositoriesCount int64  `json:"integrated_repositories_count,omitempty"`
	ScmPosturePolicyID          string `json:"scm_posture_policy_id,omitempty"`
}

// Common exposes the embedded shared fields, so any concrete SCM unit type that
// embeds ScmUnitCommonFields satisfies the generic unit framework's constraint.
func (c ScmUnitCommonFields) Common() ScmUnitCommonFields { return c }
