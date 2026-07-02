package s3_bucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"
	cc "terraform-provider-orcasecurity/orcasecurity/config_integration_common"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// arnOrURLPattern mirrors Orca's webhook schema: the value must start with an ARN prefix or
// an http/https/s3 scheme. Plain bucket names are not accepted; that prevents customers from
// pasting a bucket name and hitting a confusing connectivity error later.
var arnOrURLPattern = regexp.MustCompile(`^(arn:|https?://|s3://)`)

// errRenderingPolicy is the diagnostic summary used when the AfterExtract hook fails to render
// the bucket policy. Centralised so future copy edits land in one place.
const errRenderingPolicy = "Error rendering S3 bucket policy"

// state is the per-variant Terraform model. CommonFields carries id / template_name /
// is_enabled / is_default plus the GetCommon/SetCommon glue the generic spec needs.
type state struct {
	cc.CommonFields
	ArnOrURL         types.String `tfsdk:"arn_or_url"`
	Folder           types.String `tfsdk:"folder"`
	BucketName       types.String `tfsdk:"bucket_name"`
	UploaderRoleArn  types.String `tfsdk:"uploader_role_arn"`
	BucketPolicyJSON types.String `tfsdk:"bucket_policy_json"`
	BucketPolicyHint types.String `tfsdk:"bucket_policy_instructions"`
}

func NewS3BucketResource() resource.Resource {
	return cc.New(cc.Spec[api_client.S3BucketExternalServiceConfig]{
		TypeNameSuffix: "_integration_s3_bucket",
		UIName:         "S3 bucket integration",
		Description:    "Manage an S3 Bucket integration in Orca. Orca uploads alert exports into the customer-owned S3 bucket. After creating the resource, attach the rendered `bucket_policy_json` to the bucket so Orca's uploader role can `PutObject` under `folder/*`.",
		VariantAttributes: map[string]schema.Attribute{
			"arn_or_url": schema.StringAttribute{
				Required:    true,
				Description: "S3 bucket ARN (for example, `arn:aws:s3:::my-bucket`) or URL with an `https://`, `http://`, or `s3://` scheme. The rendered `bucket_policy_json` always uses the bucket-name-only ARN form regardless of which shape is supplied here.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(arnOrURLPattern, "must start with arn:, https://, http://, or s3://"),
				},
			},
			"folder": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Sub-folder inside the bucket where Orca writes objects. Defaults to the bucket root.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_name": schema.StringAttribute{
				Computed:      true,
				Description:   "Bucket name parsed from `arn_or_url`. Used to build `bucket_policy_json`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"uploader_role_arn": schema.StringAttribute{
				Computed:      true,
				Description:   "Orca's S3 uploader role ARN. Use this value as the `Principal.AWS` field in the bucket policy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_policy_json": schema.StringAttribute{
				Computed:      true,
				Description:   "Bucket policy JSON ready to attach to the S3 bucket. Grants Orca's uploader role `s3:PutObject` and `s3:PutObjectAcl` under `<folder>/*` and requires the `bucket-owner-full-control` ACL header.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_policy_instructions": schema.StringAttribute{
				Computed:      true,
				Description:   "Human-readable instruction string explaining what to do with `bucket_policy_json`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		NewState: func() cc.State { return &state{} },
		BuildPayload: func(_ context.Context, st cc.State, _ *diag.Diagnostics) api_client.S3BucketExternalServiceConfig {
			s := st.(*state)
			return api_client.S3BucketExternalServiceConfig{
				TemplateName: s.TemplateName.ValueString(),
				IsEnabled:    s.IsEnabled.ValueBool(),
				IsDefault:    s.IsDefault.ValueBool(),
				Config: api_client.S3BucketConfig{
					ArnOrURL: s.ArnOrURL.ValueString(),
					Folder:   s.Folder.ValueString(),
				},
			}
		},
		Extract: func(o *api_client.S3BucketExternalServiceConfig, st cc.State, _ *diag.Diagnostics) cc.APIObject {
			s := st.(*state)
			if o.Config.ArnOrURL != "" {
				s.ArnOrURL = types.StringValue(o.Config.ArnOrURL)
			}
			// Folder is optional — mirror what the API returned (possibly empty) so the computed
			// default plan modifier stays satisfied.
			s.Folder = types.StringValue(o.Config.Folder)
			return cc.APIObject{ID: o.ID, TemplateName: o.TemplateName, IsEnabled: o.IsEnabled, IsDefault: o.IsDefault}
		},
		// AfterExtract renders the computed bucket-policy fields. It needs a second endpoint
		// (GET /api/settings for the uploader role ARN), so it runs via the client-aware hook
		// rather than Extract.
		AfterExtract: populateComputed,
		Create:       (*api_client.APIClient).CreateS3BucketConfig,
		Get:          (*api_client.APIClient).GetS3BucketConfig,
		Update:       (*api_client.APIClient).UpdateS3BucketConfig,
		Delete:       (*api_client.APIClient).DeleteS3BucketConfig,
	})
}

// populateComputed derives bucket_name / uploader_role_arn / bucket_policy_json /
// bucket_policy_instructions from arn_or_url + the Orca settings document.
func populateComputed(client *api_client.APIClient, st cc.State, diags *diag.Diagnostics) {
	s := st.(*state)
	bucket, err := parseBucketName(s.ArnOrURL.ValueString())
	if err != nil {
		diags.AddError(errRenderingPolicy, err.Error())
		return
	}
	settings, err := client.GetOrcaSettings()
	if err != nil {
		diags.AddError(errRenderingPolicy, fmt.Sprintf("could not fetch Orca settings to build bucket policy: %s", err.Error()))
		return
	}
	policy, err := buildBucketPolicyJSON(bucket, s.Folder.ValueString(), settings.ReportUploaderArn)
	if err != nil {
		diags.AddError(errRenderingPolicy, err.Error())
		return
	}
	s.BucketName = types.StringValue(bucket)
	s.UploaderRoleArn = types.StringValue(settings.ReportUploaderArn)
	s.BucketPolicyJSON = types.StringValue(policy)
	s.BucketPolicyHint = types.StringValue("Add this policy to the bucket permissions.")
}

// parseBucketName extracts the bucket name from an ARN or HTTPS/S3 URL. Mirrors the logic in
// base_api/api/views/external_services/connectivity_checks._check_s3_bucket_config.
func parseBucketName(arnOrURL string) (string, error) {
	arnOrURL = strings.TrimSpace(arnOrURL)
	if arnOrURL == "" {
		return "", fmt.Errorf("arn_or_url is empty")
	}
	if strings.HasPrefix(arnOrURL, "http://") || strings.HasPrefix(arnOrURL, "https://") || strings.HasPrefix(arnOrURL, "s3://") {
		return bucketFromURL(arnOrURL)
	}
	if strings.HasPrefix(arnOrURL, "arn:") {
		parts := strings.Split(arnOrURL, ":")
		bucket := parts[len(parts)-1]
		if bucket == "" {
			return "", fmt.Errorf("could not extract bucket name from ARN %q", arnOrURL)
		}
		return strings.TrimPrefix(bucket, "/"), nil
	}
	return "", fmt.Errorf("arn_or_url must start with arn:, https://, http://, or s3://")
}

// bucketFromURL resolves the bucket name from an HTTP, HTTPS, or s3:// URL.
func bucketFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("could not parse arn_or_url as URL: %w", err)
	}
	host := parsed.Host
	if host == "" {
		return "", fmt.Errorf("could not extract bucket name from URL %q", rawURL)
	}
	// Path-style: s3.amazonaws.com/<bucket>/..., s3.<region>.amazonaws.com/<bucket>/...,
	// or legacy s3-<region>.amazonaws.com/<bucket>/... — bucket is the first path segment.
	lowerHost := strings.ToLower(host)
	if lowerHost == "s3.amazonaws.com" || strings.HasPrefix(lowerHost, "s3.") || strings.HasPrefix(lowerHost, "s3-") {
		segment := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 2)[0]
		if segment == "" {
			return "", fmt.Errorf("could not extract bucket name from path-style URL %q", rawURL)
		}
		return segment, nil
	}
	// Virtual-hosted style: <bucket>.s3.<region>.amazonaws.com — take the first label.
	if strings.Contains(host, ".") {
		return strings.Split(host, ".")[0], nil
	}
	// s3://bucket-name form.
	return host, nil
}

// buildBucketPolicyJSON renders the policy document the customer must attach to the bucket so
// Orca's uploader role can write into “folder/*“. The Resource ARN matches what Orca's
// connectivity check exercises (PutObject with bucket-owner-full-control ACL).
func buildBucketPolicyJSON(bucketName, folder, uploaderArn string) (string, error) {
	resource := fmt.Sprintf("arn:aws:s3:::%s", bucketName)
	if folder != "" {
		resource = fmt.Sprintf("%s/%s/*", resource, strings.Trim(folder, "/"))
	} else {
		resource = fmt.Sprintf("%s/*", resource)
	}

	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect":    "Allow",
				"Principal": map[string]interface{}{"AWS": uploaderArn},
				"Action":    []string{"s3:PutObject", "s3:PutObjectAcl"},
				"Resource":  resource,
				"Condition": map[string]interface{}{
					"StringEquals": map[string]interface{}{
						"s3:x-amz-acl": "bucket-owner-full-control",
					},
				},
			},
		},
	}
	encoded, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
