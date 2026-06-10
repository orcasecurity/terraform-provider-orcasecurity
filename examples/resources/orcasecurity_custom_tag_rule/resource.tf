# Using a discovery query string
resource "orcasecurity_custom_tag_rule" "string_rule" {
  name        = "Tag public EC2 instances"
  description = "Automatically tag all EC2 instances with a public IP"
  rule        = "AwsEc2Instance with PublicIps"
  tags = {
    "exposure" = "public"
  }
}

# Using a JSON discovery query (the format used by the Orca UI)
resource "orcasecurity_custom_tag_rule" "json_rule" {
  name        = "Tag internet-facing EC2 instances"
  description = "Automatically tag all internet-facing EC2 instances"
  rule_type   = "json"
  rule = jsonencode({
    type   = "object_set"
    models = ["AwsEc2Instance"]
    with = {
      type     = "operation"
      operator = "and"
      values = [
        {
          key      = "IsInternetFacing"
          values   = [true]
          type     = "bool"
          operator = "eq"
        }
      ]
    }
  })
  tags = {
    "exposure" = "internet-facing"
  }
}
