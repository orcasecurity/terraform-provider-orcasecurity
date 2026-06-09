package shift_left_policy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var policyTypes = []string{
	"iac",
	"sast",
	"file_system",
	"file_system_vulnerabilities",
	"file_system_secret_detection",
	"container_image",
	"scm_posture",
	"licenses",
	"sca",
}

type conditionsModel struct {
	FixAvailable       types.Bool     `tfsdk:"fix_available"`
	FromBaseImage      types.Bool     `tfsdk:"from_base_image"`
	DaysFromDiscovery  types.Int64    `tfsdk:"days_from_discovery"`
	DaysFromFix        types.Int64    `tfsdk:"days_from_fix"`
	HasExploit         types.Bool     `tfsdk:"has_exploit"`
	SeveritiesOperator types.String   `tfsdk:"severities_operator"`
	SeveritiesValues   []types.String `tfsdk:"severities_values"`
}

type baseControlModel struct {
	ID         types.String     `tfsdk:"id"`
	Title      types.String     `tfsdk:"title"`
	Priority   types.String     `tfsdk:"priority"`
	Disabled   types.Bool       `tfsdk:"disabled"`
	Conditions *conditionsModel `tfsdk:"conditions"`
}

type iacControlModel struct {
	baseControlModel
	Frameworks        []types.String `tfsdk:"frameworks"`
	OrcaAlertRuleType types.String   `tfsdk:"orca_alert_rule_type"`
}

type sastControlModel struct {
	baseControlModel
	Languages  []types.String `tfsdk:"languages"`
	Owasp      []types.String `tfsdk:"owasp"`
	Cwe        []types.String `tfsdk:"cwe"`
	Section    types.String   `tfsdk:"section"`
	Confidence types.String   `tfsdk:"confidence"`
	Impact     types.String   `tfsdk:"impact"`
	Likelihood types.String   `tfsdk:"likelihood"`
}

type containerControlModel struct {
	baseControlModel
	Origin types.String `tfsdk:"origin"`
}

type scmControlModel struct {
	ID       types.String   `tfsdk:"id"`
	Priority types.String   `tfsdk:"priority"`
	Disabled types.Bool     `tfsdk:"disabled"`
	Scm      types.String   `tfsdk:"scm"`
	Entity   types.String   `tfsdk:"entity"`
	Threat   []types.String `tfsdk:"threat"`
}

type licenseControlModel struct {
	baseControlModel
	LicenseID       types.String   `tfsdk:"license_id"`
	LicenseCategory types.String   `tfsdk:"license_category"`
	IsOsiApproved   types.Bool     `tfsdk:"is_osi_approved"`
	IsDeprecated    types.Bool     `tfsdk:"is_deprecated"`
	IsFsfLibre      types.Bool     `tfsdk:"is_fsf_libre"`
	Url             types.String   `tfsdk:"url"`
	AdditionalInfo  []types.String `tfsdk:"additional_info"`
}

type controlsBlockModel struct {
	AllControls types.Bool         `tfsdk:"all_controls"`
	Controls    []baseControlModel `tfsdk:"controls"`
}

type iacBlockModel struct {
	AllControls types.Bool        `tfsdk:"all_controls"`
	Controls    []iacControlModel `tfsdk:"controls"`
}

type sastBlockModel struct {
	AllControls types.Bool         `tfsdk:"all_controls"`
	Controls    []sastControlModel `tfsdk:"controls"`
}

type containerScopeBlockModel struct {
	AllControls types.Bool              `tfsdk:"all_controls"`
	Controls    []containerControlModel `tfsdk:"controls"`
}

type containerImageBlockModel struct {
	FeatureScope                []types.String            `tfsdk:"feature_scope"`
	Vulnerabilities             *containerScopeBlockModel `tfsdk:"vulnerabilities"`
	SecretDetection             *containerScopeBlockModel `tfsdk:"secret_detection"`
	ContainerImageBestPractices *containerScopeBlockModel `tfsdk:"container_image_best_practices"`
	Custom                      *containerScopeBlockModel `tfsdk:"custom"`
}

type scmScopeEntryModel struct {
	Key types.String   `tfsdk:"key"`
	Ids []types.String `tfsdk:"ids"`
}

type scmPostureBlockModel struct {
	Scope    []scmScopeEntryModel `tfsdk:"scope"`
	Controls []scmControlModel    `tfsdk:"controls"`
}

type licensesBlockModel struct {
	AllControls types.Bool            `tfsdk:"all_controls"`
	Controls    []licenseControlModel `tfsdk:"controls"`
}

type shiftLeftPolicyResourceModel struct {
	ID                       types.String   `tfsdk:"id"`
	Type                     types.String   `tfsdk:"type"`
	Name                     types.String   `tfsdk:"name"`
	Description              types.String   `tfsdk:"description"`
	Disabled                 types.Bool     `tfsdk:"disabled"`
	WarnMode                 types.Bool     `tfsdk:"warn_mode"`
	PriorityFailureThreshold types.String   `tfsdk:"priority_failure_threshold"`
	ProjectsIds              []types.String `tfsdk:"projects_ids"`
	Builtin                  types.Bool     `tfsdk:"builtin"`

	Iac                       *iacBlockModel            `tfsdk:"iac"`
	Sast                      *sastBlockModel           `tfsdk:"sast"`
	FileSystem                *controlsBlockModel       `tfsdk:"file_system"`
	FileSystemVulnerabilities *controlsBlockModel       `tfsdk:"file_system_vulnerabilities"`
	FileSystemSecretDetection *controlsBlockModel       `tfsdk:"file_system_secret_detection"`
	ContainerImage            *containerImageBlockModel `tfsdk:"container_image"`
	ScmPosture                *scmPostureBlockModel     `tfsdk:"scm_posture"`
	Licenses                  *licensesBlockModel       `tfsdk:"licenses"`
	Sca                       *licensesBlockModel       `tfsdk:"sca"`
}
