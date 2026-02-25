# Basic automation v2 with Jira Cloud using external configuration
resource "orcasecurity_automation_v2" "jira_cloud_basic" {
  name        = "Critical Alerts to Jira Cloud"
  description = "Send critical security alerts to Jira Cloud using external config"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
      with = {
        operator = "and"
        type     = "operation"
        values = [
          {
            key      = "AlertType"
            operator = "in"
            type     = "str"
            values = [
              "AWS EC2 instance allows public ingress access on NetBIOS port 137",
              "S3 bucket data is not protected",
              "SQS queue with public access",
              "Azure SQL Database with basic sku"
            ]
          },
          {
            key      = "OrcaScore"
            operator = "range"
            type     = "float"
            values   = [7, 10]
          },
          {
            keys     = ["CloudAccount"]
            models   = ["CloudAccount"]
            operator = "has"
            type     = "object"
            with = {
              key      = "CloudProvider"
              operator = "in"
              type     = "str"
              values   = ["aws", "azure"]
            }
          }
        ]
      }
    })
  }

  jira_cloud_template = {
    external_config_id = "12345678-1234-1234-1234-123456789abc"
  }
}

# Jira Cloud automation with parent issue
resource "orcasecurity_automation_v2" "jira_cloud_with_parent" {
  name        = "Security Alerts under Epic"
  description = "Create Jira tickets under a parent epic for security alerts"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  jira_cloud_template = {
    external_config_id = "12345678-1234-1234-1234-123456789abc"
    parent_issue       = "SEC-123"
  }
}

# Datadog integration with LOGS type
resource "orcasecurity_automation_v2" "datadog_logs" {
  name        = "Forward Alerts to Datadog Logs"
  description = "Send security alerts to Datadog as structured logs"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  datadog_template = {
    external_config_id = "87654321-4321-4321-4321-cba987654321"
    type               = "LOGS"
  }
}

# Datadog integration with EVENT type
resource "orcasecurity_automation_v2" "datadog_events" {
  name        = "Security Events to Datadog"
  description = "Send security alerts to Datadog as events"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  datadog_template = {
    external_config_id = "87654321-4321-4321-4321-cba987654321"
    type               = "EVENT"
  }
}

# Slack integration using external configuration
resource "orcasecurity_automation_v2" "slack_alerts" {
  name        = "Security Team Slack Notifications"
  description = "Send security alerts to Slack channel"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  slack_template = {
    external_config_id = "11111111-2222-3333-4444-555555555555"
  }
}

# Email notifications
resource "orcasecurity_automation_v2" "email_security_team" {
  name        = "Email Security Team"
  description = "Send critical alerts to security team via email"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  email_template = {
    email        = ["security@company.com", "admin@company.com"]
    multi_alerts = true
  }
}

# Email with individual alerts (not aggregated)
resource "orcasecurity_automation_v2" "email_individual" {
  name        = "Individual Alert Emails"
  description = "Send each alert as a separate email"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  email_template = {
    email        = ["incident-response@company.com"]
    multi_alerts = false
  }
}

# Snooze automation for test environments
resource "orcasecurity_automation_v2" "snooze_test_env" {
  name        = "Auto-snooze Test Environment Alerts"
  description = "Automatically snooze alerts from test environments for 7 days"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  snooze_template = {
    days          = 7
    reason        = "Test Environment"
    justification = "These alerts are from test environments and can be safely snoozed during development cycles"
  }
}

# Advanced filtering with complex sonar query
resource "orcasecurity_automation_v2" "advanced_filtering" {
  name        = "Critical AWS/Azure Alerts"
  description = "Complex filtering for specific alert types and cloud providers"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
      with = {
        operator = "and"
        type     = "operation"
        values = [
          {
            key      = "AlertType"
            operator = "in"
            type     = "str"
            values = [
              "AWS EC2 instance allows public ingress access on NetBIOS port 137",
              "S3 bucket data is not protected",
              "SQS queue with public access",
              "Azure SQL Database with basic sku"
            ]
          },
          {
            key      = "OrcaScore"
            operator = "range"
            type     = "float"
            values   = [7, 10]
          },
          {
            keys     = ["CloudAccount"]
            models   = ["CloudAccount"]
            operator = "has"
            type     = "object"
            with = {
              key      = "CloudProvider"
              operator = "in"
              type     = "str"
              values   = ["aws", "azure"]
            }
          }
        ]
      }
    })
  }

  slack_template = {
    external_config_id = "11111111-2222-3333-4444-555555555555"
  }
}

# Temporary automation with end time
resource "orcasecurity_automation_v2" "incident_response" {
  name        = "Incident Response Monitoring"
  description = "Enhanced monitoring during incident response period"
  status      = "enabled"
  end_time    = "2024-12-31T23:59:59Z"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  email_template = {
    email        = ["incident-response@company.com", "security-lead@company.com"]
    multi_alerts = false
  }
}

# Multi-service integration example
resource "orcasecurity_automation_v2" "multi_service_integration" {
  name        = "Comprehensive Security Monitoring"
  description = "Forward alerts to multiple security tools and communication channels"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  # Communication services
  slack_template = {
    external_config_id = "11111111-2222-3333-4444-555555555555"
  }

  ms_teams_template = {
    external_config_id = "22222222-3333-4444-5555-666666666666"
  }

  opsgenie_template = {
    external_config_id = "33333333-4444-5555-6666-777777777777"
  }

  # SIEM and security platforms
  splunk_template = {
    external_config_id = "44444444-5555-6666-7777-888888888888"
  }

  aws_security_hub_template = {
    external_config_id = "55555555-6666-7777-8888-999999999999"
  }

  # Ticketing systems
  jira_cloud_template = {
    external_config_id = "12345678-1234-1234-1234-123456789abc"
  }

  servicenow_incidents_template = {
    external_config_id = "66666666-7777-8888-9999-aaaaaaaaaaaa"
  }
}

# SIEM integrations
resource "orcasecurity_automation_v2" "siem_integration" {
  name        = "SIEM Data Integration"
  description = "Forward security data to multiple SIEM platforms"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  splunk_template = {
    external_config_id = "44444444-5555-6666-7777-888888888888"
  }

  chronicle_template = {
    external_config_id = "77777777-8888-9999-aaaa-bbbbbbbbbbbb"
  }

  sumo_logic_template = {
    external_config_id = "88888888-9999-aaaa-bbbb-cccccccccccc"
  }

  azure_sentinel_template = {
    external_config_id = "99999999-aaaa-bbbb-cccc-dddddddddddd"
  }
}

# Cloud services integration
resource "orcasecurity_automation_v2" "cloud_services" {
  name        = "Cloud Native Security Integration"
  description = "Integrate with cloud-native security and messaging services"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  # AWS services
  aws_security_hub_template = {
    external_config_id = "55555555-6666-7777-8888-999999999999"
  }

  aws_security_lake_template = {
    external_config_id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
  }

  aws_sqs_template = {
    external_config_id = "bbbbbbbb-cccc-dddd-eeee-ffffffffffff"
  }

  aws_sns_template = {
    external_config_id = "cccccccc-dddd-eeee-ffff-gggggggggggg"
  }

  # Google Cloud
  gcp_pub_sub_template = {
    external_config_id = "dddddddd-eeee-ffff-gggg-hhhhhhhhhhhh"
  }
}

# DevOps and project management tools
resource "orcasecurity_automation_v2" "devops_integration" {
  name        = "DevOps Security Integration"
  description = "Integrate security alerts with development and project management tools"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  # Jira integrations
  jira_cloud_template = {
    external_config_id = "12345678-1234-1234-1234-123456789abc"
  }

  jira_server_template = {
    external_config_id = "eeeeeeee-ffff-gggg-hhhh-iiiiiiiiiiii"
  }

  # Modern project management
  linear_template = {
    external_config_id = "ffffffff-gggg-hhhh-iiii-jjjjjjjjjjjj"
  }

  monday_template = {
    external_config_id = "gggggggg-hhhh-iiii-jjjj-kkkkkkkkkkkk"
  }

  # Microsoft ecosystem
  azure_devops_template = {
    external_config_id = "hhhhhhhh-iiii-jjjj-kkkk-llllllllllll"
  }
}

# Webhook and custom integrations
resource "orcasecurity_automation_v2" "custom_integrations" {
  name        = "Custom Webhook Integration"
  description = "Forward alerts to custom endpoints and specialized tools"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  webhook_template = {
    external_config_id = "iiiiiiii-jjjj-kkkk-llll-mmmmmmmmmmmm"
  }

  tines_template = {
    external_config_id = "jjjjjjjj-kkkk-llll-mmmm-nnnnnnnnnnnn"
  }

  torq_template = {
    external_config_id = "kkkkkkkk-llll-mmmm-nnnn-oooooooooooo"
  }
}

# Data analytics and monitoring
resource "orcasecurity_automation_v2" "analytics_integration" {
  name        = "Security Analytics Integration"
  description = "Forward security data to analytics and monitoring platforms"
  status      = "enabled"

  filter = {
    sonar_query = jsonencode({
      models = ["Alert"]
      type   = "object_set"
    })
  }

  datadog_template = {
    external_config_id = "87654321-4321-4321-4321-cba987654321"
    type               = "LOGS"
  }

  cribl_template = {
    external_config_id = "llllllll-mmmm-nnnn-oooo-pppppppppppp"
  }

  snowflake_template = {
    external_config_id = "mmmmmmmm-nnnn-oooo-pppp-qqqqqqqqqqqq"
  }
}
