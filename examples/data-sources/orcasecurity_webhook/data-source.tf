data "orcasecurity_webhook" "example" {
  name = "the-webhook-name" //as shown in Webhooks popup: https://app.orcasecurity.io/integrations?configure=%22webhookModalId%22
}

# custom_headers maps a header name to a list of { custom = "<value>" } objects.
# Secret headers show "__orca_secret_set__" instead of their value.
output "example_header" {
  value = data.orcasecurity_webhook.example.config.custom_headers["X-Header"][0].custom
}
