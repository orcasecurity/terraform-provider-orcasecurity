terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

resource "orcasecurity_custom_widget" "closed_alerts_30_days_widget" {
  name               = "Closed Alerts Last 30 Days by Risk Level"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Neglected assets",
                  "Vendor services misconfigurations",
                  "Workload misconfigurations",
                  "Best practices",
                  "Data protection",
                  "Data at risk",
                  "IAM misconfigurations",
                  "Network misconfigurations",
                  "Logging and monitoring",
                  "Authentication",
                  "Lateral movement",
                  "Vulnerabilities",
                  "Malware",
                  "Malicious activity",
                  "System integrity",
                  "Suspicious activity"
                ],
                "operator" : "in"
              },
              {
                "key" : "Status",
                "type" : "str",
                "values" : [
                  "closed"
                ],
                "operator" : "in"
              },
              {
                "key" : "ClosedTime",
                "values" : [
                  30
                ],
                "type" : "datetime",
                "operator" : "in_past",
                "value_type" : "days"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "RiskLevel"
        ],
        group_by_list = [
          "RiskLevel"
        ]
      }
      field = {
        name = "RiskLevel",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "closed_alerts_30_days_category_widget" {
  name               = "Closed Alerts Last 30 Days by Category"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Neglected assets",
                  "Vendor services misconfigurations",
                  "Workload misconfigurations",
                  "Best practices",
                  "Data protection",
                  "Data at risk",
                  "IAM misconfigurations",
                  "Network misconfigurations",
                  "Logging and monitoring",
                  "Authentication",
                  "Lateral movement",
                  "Vulnerabilities",
                  "Malware",
                  "Malicious activity",
                  "System integrity",
                  "Suspicious activity"
                ],
                "operator" : "in"
              },
              {
                "key" : "Status",
                "type" : "str",
                "values" : [
                  "closed"
                ],
                "operator" : "in"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "Category"
        ],
        group_by_list = [
          "Category"
        ]
      }
      field = {
        name = "Category",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "closed_alerts_30_days_account_widget" {
  name               = "Closed Alerts Last 30 Days by Account"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Neglected assets",
                  "Vendor services misconfigurations",
                  "Workload misconfigurations",
                  "Best practices",
                  "Data protection",
                  "Data at risk",
                  "IAM misconfigurations",
                  "Network misconfigurations",
                  "Logging and monitoring",
                  "Authentication",
                  "Lateral movement",
                  "Vulnerabilities",
                  "Malware",
                  "Malicious activity",
                  "System integrity",
                  "Suspicious activity"
                ],
                "operator" : "in"
              },
              {
                "key" : "Status",
                "type" : "str",
                "values" : [
                  "closed"
                ],
                "operator" : "in"
              },
              {
                "key" : "ClosedTime",
                "values" : [
                  30
                ],
                "type" : "datetime",
                "operator" : "in_past",
                "value_type" : "days"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "CloudAccount.Name"
        ],
        group_by_list = [
          "CloudAccount.Name"
        ]
      }
      field = {
        name = "CloudAccount.Name",
        type = "str"
      }
    }
  }
}

output "closed_alerts_30_days_widget_id" {
  value = orcasecurity_custom_widget.closed_alerts_30_days_widget.id
}

output "closed_alerts_30_days_category_widget_id" {
  value = orcasecurity_custom_widget.closed_alerts_30_days_category_widget.id
}

output "closed_alerts_30_days_account_widget_id" {
  value = orcasecurity_custom_widget.closed_alerts_30_days_account_widget.id
}