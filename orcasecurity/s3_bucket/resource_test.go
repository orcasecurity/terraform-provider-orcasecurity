package s3_bucket

import (
	"encoding/json"
	"testing"
)

func TestParseBucketName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "virtual-hosted style",
			input: "https://my-bucket.s3.us-east-1.amazonaws.com",
			want:  "my-bucket",
		},
		{
			name:  "virtual-hosted bucket name containing us-gov-",
			input: "https://us-gov-reports.s3.amazonaws.com",
			want:  "us-gov-reports",
		},
		{
			name:  "virtual-hosted style with path",
			input: "https://my-bucket.s3.us-east-1.amazonaws.com/folder/key",
			want:  "my-bucket",
		},
		{
			name:  "path-style global endpoint",
			input: "https://s3.amazonaws.com/my-bucket/obj",
			want:  "my-bucket",
		},
		{
			name:  "path-style regional endpoint",
			input: "https://s3.us-west-2.amazonaws.com/customer-bucket",
			want:  "customer-bucket",
		},
		{
			name:  "path-style legacy regional endpoint",
			input: "https://s3-us-west-2.amazonaws.com/customer-bucket/obj",
			want:  "customer-bucket",
		},
		{
			name:  "path-style dualstack endpoint",
			input: "https://s3.dualstack.us-east-1.amazonaws.com/my-bucket",
			want:  "my-bucket",
		},
		{
			name:    "path-style without bucket segment",
			input:   "https://s3.us-west-2.amazonaws.com/",
			wantErr: true,
		},
		{
			name:  "s3 scheme",
			input: "s3://my-bucket/key",
			want:  "my-bucket",
		},
		{
			name:  "arn",
			input: "arn:aws:s3:::my-bucket",
			want:  "my-bucket",
		},
		{
			name:  "govcloud arn",
			input: "arn:aws-us-gov:s3:::my-bucket-name",
			want:  "my-bucket-name",
		},
		{
			name:  "china arn",
			input: "arn:aws-cn:s3:::my-bucket-name",
			want:  "my-bucket-name",
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "unsupported prefix",
			input:   "ftp://example.com/bucket",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseBucketName(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got bucket %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("parseBucketName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPolicyPartition(t *testing.T) {
	govUploader := "arn:aws-us-gov:iam::247207723357:role/integrations_s3_uploader"
	commercialUploader := "arn:aws:iam::123456789012:role/integrations_s3_uploader"

	tests := []struct {
		name              string
		resourcePartition string
		uploader          string
		want              string
	}{
		{
			name:              "settings resource_partition wins",
			resourcePartition: "aws-us-gov",
			uploader:          commercialUploader,
			want:              "aws-us-gov",
		},
		{
			name:              "empty settings falls back to uploader",
			resourcePartition: "",
			uploader:          govUploader,
			want:              "aws-us-gov",
		},
		{
			name:              "uppercase settings partition is normalized",
			resourcePartition: "AWS-US-GOV",
			uploader:          commercialUploader,
			want:              "aws-us-gov",
		},
		{
			name:              "bogus settings partition falls back to uploader",
			resourcePartition: "bogus",
			uploader:          govUploader,
			want:              "aws-us-gov",
		},
		{
			name:              "bogus settings and uploader default to aws",
			resourcePartition: "bogus",
			uploader:          "arn:bogus:iam::123456789012:role/integrations_s3_uploader",
			want:              "aws",
		},
		{
			name:              "china settings",
			resourcePartition: "aws-cn",
			uploader:          "arn:aws-cn:iam::123456789012:role/integrations_s3_uploader",
			want:              "aws-cn",
		},
		{
			name:              "empty both default to aws",
			resourcePartition: "",
			uploader:          "",
			want:              "aws",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := policyPartition(tc.resourcePartition, tc.uploader); got != tc.want {
				t.Errorf("policyPartition(%q, %q) = %q, want %q", tc.resourcePartition, tc.uploader, got, tc.want)
			}
		})
	}
}

func TestBuildBucketPolicyJSON_UsesOrcaTenantPartition(t *testing.T) {
	govUploader := "arn:aws-us-gov:iam::247207723357:role/integrations_s3_uploader"
	commercialUploader := "arn:aws:iam::123456789012:role/integrations_s3_uploader"

	tests := []struct {
		name              string
		bucket            string
		folder            string
		uploader          string
		resourcePartition string
		wantResource      string
	}{
		{
			name:              "commercial settings",
			bucket:            "my-bucket",
			uploader:          commercialUploader,
			resourcePartition: "aws",
			wantResource:      "arn:aws:s3:::my-bucket/*",
		},
		{
			name:              "govcloud settings",
			bucket:            "my-bucket-name",
			uploader:          govUploader,
			resourcePartition: "aws-us-gov",
			wantResource:      "arn:aws-us-gov:s3:::my-bucket-name/*",
		},
		{
			name:              "govcloud settings with folder",
			bucket:            "my-bucket-name",
			folder:            "exports",
			uploader:          govUploader,
			resourcePartition: "aws-us-gov",
			wantResource:      "arn:aws-us-gov:s3:::my-bucket-name/exports/*",
		},
		{
			name:         "uploader fallback when settings partition empty",
			bucket:       "my-bucket-name",
			uploader:     govUploader,
			wantResource: "arn:aws-us-gov:s3:::my-bucket-name/*",
		},
		{
			name:              "us-gov bucket name on commercial tenant stays aws",
			bucket:            "us-gov-reports",
			uploader:          commercialUploader,
			resourcePartition: "aws",
			wantResource:      "arn:aws:s3:::us-gov-reports/*",
		},
		{
			name:              "settings partition wins over commercial-looking uploader",
			bucket:            "my-bucket-name",
			uploader:          commercialUploader,
			resourcePartition: "aws-us-gov",
			wantResource:      "arn:aws-us-gov:s3:::my-bucket-name/*",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildBucketPolicyJSON(tc.bucket, tc.folder, tc.uploader, tc.resourcePartition)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resource := policyResourceARN(t, got); resource != tc.wantResource {
				t.Errorf("Resource = %q, want %q", resource, tc.wantResource)
			}
		})
	}
}

func policyResourceARN(t *testing.T, policyJSON string) string {
	t.Helper()
	var doc struct {
		Statement []struct {
			Resource string `json:"Resource"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(policyJSON), &doc); err != nil {
		t.Fatalf("unmarshal policy: %v", err)
	}
	if len(doc.Statement) != 1 {
		t.Fatalf("got %d statements, want 1", len(doc.Statement))
	}
	return doc.Statement[0].Resource
}
