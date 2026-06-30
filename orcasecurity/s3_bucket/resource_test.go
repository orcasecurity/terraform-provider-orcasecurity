package s3_bucket

import "testing"

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
