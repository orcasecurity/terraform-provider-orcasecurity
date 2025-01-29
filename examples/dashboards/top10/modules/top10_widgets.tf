terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

resource "orcasecurity_custom_widget" "malware_alerts_widget" {
  name               = "Top 10 Malware Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Malware"
                ],
                "operator": "in"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "vulnerability_alerts_widget" {
  name               = "Top 10 Vulnerability Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Vulnerabilities"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "data_risk_alerts_widget" {
  name               = "Top 10 Data at Risk Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Data at risk"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "iam_alerts_widget" {
  name               = "Top 10 IAM Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "IAM misconfigurations"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "authentication_alerts_widget" {
  name               = "Top 10 Authentication Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Authentication"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "lateral_movement_alerts_widget" {
  name = "Top 10 Lateral Movement Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Lateral movement"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "suspicious_activity_alerts_widget" {
  name = "Top 10 Suspicious Activity Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Suspicious activity"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "malicious_activity_alerts_widget" {
  name = "Top 10 Malicious Activity Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Malicious activity"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "data_protection_alerts_widget" {
  name = "Top 10 Data Protection Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Data protection"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "logging_monitoring_alerts_widget" {
  name = "Top 10 Logging and Monitoring Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Logging and monitoring"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "neglected_assets_alerts_widget" {
  name = "Top 10 Neglected Assets Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Neglected assets"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "network_misconfiguration_alerts_widget" {
  name = "Top 10 Network Misconfiguration Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "Network misconfigurations"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "ai_security_alerts_widget" {
  name = "Top 10 AI Security Alerts"
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
          "limit": 10,
          "models": [
            "Alert"
          ],
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
                      "Source: AI Security"
                    ],
                    "operator": "in"
                  }
                ],
                "operator": "any_match"
              }
            ],
            "operator": "and"
          }
        }),
        
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "api_security_alerts_widget" {
  name = "Top 10 API Security Alerts"
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
          "limit": 10,
          "models": [
            "Alert"
          ],
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
                      "source: apisec"
                    ],
                    "operator": "in"
                  }
                ],
                "operator": "any_match"
              }
            ],
            "operator": "and"
          }
        }),
        
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "file_integrity_alerts_widget" {
  name = "Top 10 File Integrity Alerts"
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
          "models": [
            "Alert"
          ],
          "type": "object_set",
          "with": {
            "operator": "and",
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
                "key": "Category",
                "type": "str",
                "values": [
                  "System integrity"
                ],
                "operator": "in"
              },
              {
                "keys": [
                  "CloudAccount"
                ],
                "type": "object",
                "with": {
                  "type": "operation",
                  "values": [
                    {
                      "key": "CloudProvider",
                      "type": "str",
                      "values": [
                        "shiftleft"
                      ],
                      "operator": "not_eq"
                    }
                  ],
                  "operator": "and"
                },
                "model": "CloudAccount",
                "operator": "has"
              }
            ],
        }})
        group_by : [
          "Name"
        ]
        start_at_index = 0
        order_by = ["Score"]
        limit = 10
        enable_pagination = true
      }
    }
  }
}

output "malware_alerts_widget_id" {
  value = orcasecurity_custom_widget.malware_alerts_widget.id
}

output "vulnerability_alerts_widget_id" {
  value = orcasecurity_custom_widget.vulnerability_alerts_widget.id
}

output "data_risk_alerts_widget_id" {
  value = orcasecurity_custom_widget.data_risk_alerts_widget.id
}

output "iam_alerts_widget_id" {
  value = orcasecurity_custom_widget.iam_alerts_widget.id
}

output "authentication_alerts_widget_id" {
  value = orcasecurity_custom_widget.authentication_alerts_widget.id
}

output "lateral_movement_alerts_widget_id" {
  value = orcasecurity_custom_widget.lateral_movement_alerts_widget.id
}

output "suspicious_activity_alerts_widget_id" {
  value = orcasecurity_custom_widget.suspicious_activity_alerts_widget.id
}

output "malicious_activity_alerts_widget_id" {
  value = orcasecurity_custom_widget.malicious_activity_alerts_widget.id
}

output "neglected_assets_alerts_widget_id" {
  value = orcasecurity_custom_widget.neglected_assets_alerts_widget.id
}

output "network_alerts_widget_id" {
  value = orcasecurity_custom_widget.network_misconfiguration_alerts_widget.id
}

output "ai_security_alerts_widget_id" {
  value = orcasecurity_custom_widget.ai_security_alerts_widget.id
}

output "api_security_alerts_widget_id" {
  value = orcasecurity_custom_widget.api_security_alerts_widget.id
}

output "data_protection_alerts_widget_id" {
  value = orcasecurity_custom_widget.data_protection_alerts_widget.id
}

output "file_integrity_alerts_widget_id" {
  value = orcasecurity_custom_widget.file_integrity_alerts_widget.id
}

output "logging_monitoring_alerts_widget_id" {
  value = orcasecurity_custom_widget.logging_monitoring_alerts_widget.id
}