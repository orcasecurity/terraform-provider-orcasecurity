terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

resource "orcasecurity_custom_widget" "malware_event_monitoring_widget" {
  name               = "Malware and Event Monitoring"
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
          "keys": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "key": "Category",
            "type": "str",
            "values": [
              "Malicious activity",
              "Suspicious activity",
              "Malware"
            ],
            "operator": "in"
          },
          "models": [
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

resource "orcasecurity_custom_widget" "real_time_sensor_widget" {
  name               = "Alerts from the Orca Sensor"
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
          "keys": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "type": "operation",
            "values": [
              {
                "key": "AlertSource",
                "type": "str",
                "values": [
                  "Orca Sensor"
                ],
                "operator": "in"
              },
              {
                "key": "Status",
                "type": "str",
                "values": [
                  "open",
                  "in_progress"
                ],
                "operator": "in"
              }
            ],
            "operator": "and"
          },
          "models": [
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

resource "orcasecurity_custom_widget" "cspm_secure_config_widget" {
  name               = "CSPM - Secure Configuration"
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
          "type": "object_set",
          "with": {
            "type": "operation",
            "values": [
              {
                "key": "Status",
                "type": "str",
                "values": [
                  "open",
                  "in_progress"
                ],
                "operator": "in"
              },
              {
                "key": "Labels",
                "type": "list",
                "values": [
                  {
                    "key": "Labels",
                    "type": "str",
                    "values": [
                      "CSPM"
                    ],
                    "operator": "in"
                  }
                ],
                "operator": "any_match"
              }
            ],
            "operator": "and"
          },
          "models": [
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

resource "orcasecurity_custom_widget" "top_cspm_alerts_widget" {
  name               = "Top CSPM Alerts"
  organization_level = true
  extra_params = {
    type                = "alert-table",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset",
        "status"
      ]
      request_params = {
        query = jsonencode({
          "type": "object_set",
          "with": {
            "type": "operation",
            "values": [
              {
                "key": "Status",
                "type": "str",
                "values": [
                  "open",
                  "in_progress"
                ],
                "operator": "in"
              },
              {
                "key": "Labels",
                "type": "list",
                "values": [
                  {
                    "key": "Labels",
                    "type": "str",
                    "values": [
                      "CSPM"
                    ],
                    "operator": "in"
                  }
                ],
                "operator": "any_match"
              }
            ],
            "operator": "and"
          },
          "models": [
            "Alert"
          ]
        })
        group_by : [
          "Score"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "top_sensor_alerts_widget" {
  name               = "Top Real Time Sensor Alerts"
  organization_level = true
  extra_params = {
    type                = "alert-table",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset",
        "tickets"
      ]
      request_params = {
        query = jsonencode({
          "keys": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "type": "operation",
            "values": [
              {
                "key": "AlertSource",
                "type": "str",
                "values": [
                  "Orca Sensor"
                ],
                "operator": "in"
              },
              {
                "key": "Status",
                "type": "str",
                "values": [
                  "open",
                  "in_progress"
                ],
                "operator": "in"
              }
            ],
            "operator": "and"
          },
          "models": [
            "Alert"
          ]
        })
        group_by : [
          "Score"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}


output "malware_event_monitoring_widget_id" {
  value = orcasecurity_custom_widget.malware_event_monitoring_widget.id
}

output "real_time_sensor_widget_id" {
  value = orcasecurity_custom_widget.real_time_sensor_widget.id
}

output "cspm_secure_config_widget_id" {
  value = orcasecurity_custom_widget.cspm_secure_config_widget.id
}

output "top_cspm_alerts_widget_id" {
  value = orcasecurity_custom_widget.top_cspm_alerts_widget.id
}

output "top_sensor_alerts_widget_id" {
  value = orcasecurity_custom_widget.top_sensor_alerts_widget.id
}

