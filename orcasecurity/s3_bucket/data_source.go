package s3_bucket

import (
	"context"
	"fmt"
	"terraform-provider-orcasecurity/orcasecurity/api_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &s3BucketPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &s3BucketPolicyDataSource{}
)

type s3BucketPolicyDataSource struct {
	apiClient *api_client.APIClient
}

type s3BucketPolicyDataSourceModel struct {
	ArnOrURL          types.String `tfsdk:"arn_or_url"`
	Folder            types.String `tfsdk:"folder"`
	BucketName        types.String `tfsdk:"bucket_name"`
	UploaderRoleArn   types.String `tfsdk:"uploader_role_arn"`
	BucketPolicyJSON  types.String `tfsdk:"bucket_policy_json"`
	BucketPolicyHint  types.String `tfsdk:"bucket_policy_instructions"`
}

func NewS3BucketPolicyDataSource() datasource.DataSource {
	return &s3BucketPolicyDataSource{}
}

func (ds *s3BucketPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_s3_bucket_policy"
}

func (ds *s3BucketPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ds.apiClient = req.ProviderData.(*api_client.APIClient)
}

func (ds *s3BucketPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Render the bucket policy Orca needs you to attach to an S3 bucket *before* creating an `orcasecurity_integration_s3_bucket` resource. Use this data source to break the chicken-and-egg cycle: Orca's create call runs a connectivity check that writes a test object to the bucket, so the policy has to exist first.",
		Attributes: map[string]schema.Attribute{
			"arn_or_url": schema.StringAttribute{
				Required:    true,
				Description: "S3 bucket reference. Must start with `arn:`, `https://`, `http://`, or `s3://`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(arnOrURLPattern, "must start with arn:, https://, http://, or s3://"),
				},
			},
			"folder": schema.StringAttribute{
				Optional:    true,
				Description: "Sub-folder inside the bucket. Leave unset for the bucket root.",
			},
			"bucket_name": schema.StringAttribute{
				Computed:    true,
				Description: "Bucket name parsed from `arn_or_url`.",
			},
			"uploader_role_arn": schema.StringAttribute{
				Computed:    true,
				Description: "Orca's S3 uploader role ARN (fetched from `GET /api/settings`). This is the `Principal.AWS` field in the rendered policy.",
			},
			"bucket_policy_json": schema.StringAttribute{
				Computed:    true,
				Description: "Bucket policy JSON ready to attach to the S3 bucket. Feed straight into `aws_s3_bucket_policy.policy`.",
			},
			"bucket_policy_instructions": schema.StringAttribute{
				Computed:    true,
				Description: "Human-readable instruction string: `\"Add this policy to the bucket permissions.\"`.",
			},
		},
	}
}

func (ds *s3BucketPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state s3BucketPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, err := parseBucketName(state.ArnOrURL.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid arn_or_url", err.Error())
		return
	}

	settings, err := ds.apiClient.GetOrcaSettings()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error fetching Orca settings",
			fmt.Sprintf("Could not fetch Orca settings to build bucket policy: %s", err.Error()),
		)
		return
	}

	policy, err := buildBucketPolicyJSON(bucket, state.Folder.ValueString(), settings.ReportUploaderArn)
	if err != nil {
		resp.Diagnostics.AddError("Error rendering bucket policy", err.Error())
		return
	}

	state.BucketName = types.StringValue(bucket)
	state.UploaderRoleArn = types.StringValue(settings.ReportUploaderArn)
	state.BucketPolicyJSON = types.StringValue(policy)
	state.BucketPolicyHint = types.StringValue("Add this policy to the bucket permissions.")

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
