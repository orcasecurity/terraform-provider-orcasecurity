package s3_bucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// arnOrURLPattern mirrors Orca's webhook schema: the value must start with an ARN prefix or
// an http/https/s3 scheme. Plain bucket names are not accepted; that prevents customers from
// pasting a bucket name and hitting a confusing connectivity error later.
var arnOrURLPattern = regexp.MustCompile(`^(arn:|https?://|s3://)`)

var (
	_ resource.Resource                = &s3BucketResource{}
	_ resource.ResourceWithConfigure   = &s3BucketResource{}
	_ resource.ResourceWithImportState = &s3BucketResource{}
)

type s3BucketResource struct {
	apiClient *api_client.APIClient
}

type s3BucketResourceModel struct {
	ID               types.String `tfsdk:"id"`
	TemplateName     types.String `tfsdk:"template_name"`
	ArnOrURL         types.String `tfsdk:"arn_or_url"`
	Folder           types.String `tfsdk:"folder"`
	IsEnabled        types.Bool   `tfsdk:"is_enabled"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	BucketName       types.String `tfsdk:"bucket_name"`
	UploaderRoleArn  types.String `tfsdk:"uploader_role_arn"`
	BucketPolicyJSON types.String `tfsdk:"bucket_policy_json"`
	BucketPolicyHint types.String `tfsdk:"bucket_policy_instructions"`
}

func NewS3BucketResource() resource.Resource {
	return &s3BucketResource{}
}

func (r *s3BucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_s3_bucket"
}

func (r *s3BucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (r *s3BucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage an S3 Bucket integration in Orca. Orca uploads alert exports into the customer-owned S3 bucket. After creating the resource, attach the rendered `bucket_policy_json` to the bucket so Orca's uploader role can `PutObject` under `folder/*`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Orca external service config identifier (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"template_name": schema.StringAttribute{
				Required:    true,
				Description: "Template name for the S3 bucket integration. Acts as the human-readable identifier for the integration in Orca. Changing this forces a new resource.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"arn_or_url": schema.StringAttribute{
				Required:    true,
				Description: "S3 bucket ARN (for example, `arn:aws:s3:::my-bucket`) or URL with an `https://`, `http://`, or `s3://` scheme. The rendered `bucket_policy_json` always uses the bucket-name-only ARN form regardless of which shape is supplied here.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(arnOrURLPattern, "must start with arn:, https://, http://, or s3://"),
				},
			},
			"folder": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Sub-folder inside the bucket where Orca writes objects. Defaults to the bucket root.",
				Default:     nil,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the S3 bucket integration is enabled. Defaults to `true`.",
				Default:     booldefault.StaticBool(true),
			},
			"is_default": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this integration is the organisation's default S3 bucket configuration. Defaults to `false`.",
				Default:     booldefault.StaticBool(false),
			},
			"bucket_name": schema.StringAttribute{
				Computed:    true,
				Description: "Bucket name parsed from `arn_or_url`. Used to build `bucket_policy_json`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uploader_role_arn": schema.StringAttribute{
				Computed:    true,
				Description: "Orca's S3 uploader role ARN. Use this value as the `Principal.AWS` field in the bucket policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket_policy_json": schema.StringAttribute{
				Computed:    true,
				Description: "Bucket policy JSON ready to attach to the S3 bucket. Grants Orca's uploader role `s3:PutObject` and `s3:PutObjectAcl` under `<folder>/*` and requires the `bucket-owner-full-control` ACL header.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket_policy_instructions": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable instruction string explaining what to do with `bucket_policy_json`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// parseBucketName extracts the bucket name from an ARN or HTTPS/S3 URL. Mirrors the logic in
// base_api/api/views/external_services/connectivity_checks._check_s3_bucket_config.
func parseBucketName(arnOrURL string) (string, error) {
	arnOrURL = strings.TrimSpace(arnOrURL)
	if arnOrURL == "" {
		return "", fmt.Errorf("arn_or_url is empty")
	}
	if strings.HasPrefix(arnOrURL, "http://") || strings.HasPrefix(arnOrURL, "https://") || strings.HasPrefix(arnOrURL, "s3://") {
		parsed, err := url.Parse(arnOrURL)
		if err != nil {
			return "", fmt.Errorf("could not parse arn_or_url as URL: %w", err)
		}
		host := parsed.Host
		if host == "" {
			return "", fmt.Errorf("could not extract bucket name from URL %q", arnOrURL)
		}
		// Virtual-hosted style: <bucket>.s3.<region>.amazonaws.com — take the first label.
		// Path-style: s3.amazonaws.com/<bucket>/... — fall back to the first path segment.
		if strings.Contains(host, ".") {
			return strings.Split(host, ".")[0], nil
		}
		// s3://bucket-name form.
		return host, nil
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

func (r *s3BucketResource) buildPayload(plan s3BucketResourceModel) api_client.S3BucketExternalServiceConfig {
	return api_client.S3BucketExternalServiceConfig{
		TemplateName: plan.TemplateName.ValueString(),
		IsEnabled:    plan.IsEnabled.ValueBool(),
		IsDefault:    plan.IsDefault.ValueBool(),
		Config: api_client.S3BucketConfig{
			ArnOrURL: plan.ArnOrURL.ValueString(),
			Folder:   plan.Folder.ValueString(),
		},
	}
}

func (r *s3BucketResource) populateComputed(plan *s3BucketResourceModel) error {
	bucket, err := parseBucketName(plan.ArnOrURL.ValueString())
	if err != nil {
		return err
	}
	settings, err := r.apiClient.GetOrcaSettings()
	if err != nil {
		return fmt.Errorf("could not fetch Orca settings to build bucket policy: %w", err)
	}
	policy, err := buildBucketPolicyJSON(bucket, plan.Folder.ValueString(), settings.ReportUploaderArn)
	if err != nil {
		return err
	}
	plan.BucketName = types.StringValue(bucket)
	plan.UploaderRoleArn = types.StringValue(settings.ReportUploaderArn)
	plan.BucketPolicyJSON = types.StringValue(policy)
	plan.BucketPolicyHint = types.StringValue("Add this policy to the bucket permissions.")
	return nil
}

func (r *s3BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.apiClient == nil {
		resp.Diagnostics.AddError("Error creating S3 bucket integration", "API client not configured.")
		return
	}

	var plan s3BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.apiClient.CreateS3BucketConfig(r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating S3 bucket integration",
			fmt.Sprintf("Could not create S3 bucket integration: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.IsEnabled = types.BoolValue(created.IsEnabled)
	plan.IsDefault = types.BoolValue(created.IsDefault)
	if created.TemplateName != "" {
		plan.TemplateName = types.StringValue(created.TemplateName)
	}
	if created.Config.ArnOrURL != "" {
		plan.ArnOrURL = types.StringValue(created.Config.ArnOrURL)
	}
	// Folder is optional — if the user omitted it, mirror what the API returned (which may be
	// an empty string) into state so the computed default plan modifier stays satisfied.
	plan.Folder = types.StringValue(created.Config.Folder)

	if err := r.populateComputed(&plan); err != nil {
		resp.Diagnostics.AddError("Error rendering S3 bucket policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state s3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.apiClient.GetS3BucketConfig(state.TemplateName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading S3 bucket integration",
			fmt.Sprintf("Could not read S3 bucket integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	if current == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(current.ID)
	state.TemplateName = types.StringValue(current.TemplateName)
	state.IsEnabled = types.BoolValue(current.IsEnabled)
	state.IsDefault = types.BoolValue(current.IsDefault)
	if current.Config.ArnOrURL != "" {
		state.ArnOrURL = types.StringValue(current.Config.ArnOrURL)
	}
	state.Folder = types.StringValue(current.Config.Folder)

	if err := r.populateComputed(&state); err != nil {
		resp.Diagnostics.AddError("Error rendering S3 bucket policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *s3BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan s3BucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state s3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.apiClient.UpdateS3BucketConfig(state.TemplateName.ValueString(), r.buildPayload(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating S3 bucket integration",
			fmt.Sprintf("Could not update S3 bucket integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(updated.ID)
	plan.IsEnabled = types.BoolValue(updated.IsEnabled)
	plan.IsDefault = types.BoolValue(updated.IsDefault)
	if updated.TemplateName != "" {
		plan.TemplateName = types.StringValue(updated.TemplateName)
	}
	if updated.Config.ArnOrURL != "" {
		plan.ArnOrURL = types.StringValue(updated.Config.ArnOrURL)
	}
	plan.Folder = types.StringValue(updated.Config.Folder)

	if err := r.populateComputed(&plan); err != nil {
		resp.Diagnostics.AddError("Error rendering S3 bucket policy", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *s3BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state s3BucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.apiClient.DeleteS3BucketConfig(state.TemplateName.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting S3 bucket integration",
			fmt.Sprintf("Could not delete S3 bucket integration %s: %s", state.TemplateName.ValueString(), err.Error()),
		)
		return
	}
}

func (r *s3BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_name"), req.ID)...)
}
