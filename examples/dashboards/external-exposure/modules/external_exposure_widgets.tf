terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

resource "orcasecurity_custom_widget" "public_facing_assets_widget" {
  name               = "Public Facing Assets by Asset Risk Score"
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

resource "orcasecurity_custom_widget" "vm_public_ingress_widget" {
  name               = "Virtual Machines with Public Ingress Ports"
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
          "keys" : [
            "Alert"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Network misconfigurations"
                ],
                "operator" : "in"
              },
              {
                "keys" : [
                  "Inventory"
                ],
                "type" : "object_set",
                "with" : {
                  "key" : "NewSubCategory",
                  "type" : "str",
                  "values" : [
                    "Virtual Instances"
                  ],
                  "operator" : "in"
                },
                "models" : [
                  "Inventory"
                ],
                "operator" : "has"
              },
              {
                "type" : "operation",
                "values" : [
                  {
                    "key" : "AlertType",
                    "type" : "str",
                    "values" : [
                      "0.0.0.0"
                    ],
                    "operator" : "containing"
                  },
                  {
                    "key" : "AlertType",
                    "type" : "str",
                    "values" : [
                      "public"
                    ],
                    "operator" : "containing"
                  },
                  {
                    "key" : "AlertType",
                    "type" : "str",
                    "values" : [
                      "unrestricted"
                    ],
                    "operator" : "containing"
                  },
                  {
                    "key" : "AlertType",
                    "type" : "str",
                    "values" : [
                      "common"
                    ],
                    "operator" : "containing"
                  }
                ],
                "operator" : "or"
              },
              {
                "key" : "AlertType",
                "type" : "str",
                "values" : [
                  "ingress"
                ],
                "operator" : "containing"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "Inventory.Type"
        ],
        group_by_list = [
          "Inventory.Type"
        ]
      }
      field = {
        name = "Inventory.Type",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "public_facing_neglected_compute_widget" {
  name               = "Public Facing Neglected Compute Alerts"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "Public facing compute assets with a large amount of vulnerabilities",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "keys" : [
            "Alert"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Neglected assets"
                ],
                "operator" : "in"
              },
              {
                "keys" : [
                  "Inventory"
                ],
                "type" : "object_set",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "key" : "Exposure",
                      "type" : "str",
                      "values" : [
                        "public_facing"
                      ],
                      "operator" : "in"
                    },
                    {
                      "key" : "NewCategory",
                      "type" : "str",
                      "values" : [
                        "Compute Services"
                      ],
                      "operator" : "in"
                    }
                  ],
                  "operator" : "and"
                },
                "models" : [
                  "Inventory"
                ],
                "operator" : "has"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "Inventory.Type"
        ],
        group_by_list = [
          "Inventory.Type"
        ]
      }
      field = {
        name = "Inventory.Type",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "attack_paths_public_widget" {
  name               = "Alerts with Attack Paths from Public Assets"
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
        "numberOfAffectedAttackPaths"
      ]
      request_params = {
        query = jsonencode({
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "Status",
                "type" : "str",
                "values" : [
                  "open",
                  "in_progress"
                ],
                "operator" : "in"
              },
              {
                "keys" : [
                  "AttackPathPrioritization"
                ],
                "type" : "object_set",
                "models" : [
                  "AttackPathPrioritizedAlert"
                ],
                "operator" : "has"
              },
              {
                "keys" : [
                  "Inventory"
                ],
                "type" : "object_set",
                "with" : {
                  "key" : "Exposure",
                  "type" : "str",
                  "values" : [
                    "public_facing"
                  ],
                  "operator" : "in"
                },
                "models" : [
                  "Inventory"
                ],
                "operator" : "has"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "Score"
        ]
        start_at_index    = 0
        order_by          = ["Score"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "public_assets_priority_risk_widget" {
  name               = "Public Facing Assets with Critical and High Risk by Asset Score"
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
                "key" : "Exposure",
                "type" : "str",
                "values" : [
                  "public_facing"
                ],
                "operator" : "in"
              },
              {
                "key" : "NewCategory",
                "type" : "str",
                "values" : [
                  "Kubernetes",
                  "Data Storage",
                  "Compute Services",
                  "AI And Machine Learning",
                  "CI Source"
                ],
                "operator" : "in"
              },
              {
                "key" : "RiskLevel",
                "type" : "str",
                "values" : [
                  "critical",
                  "high"
                ],
                "operator" : "in"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Inventory"
          ]
        })
        group_by : [
          "Inventory.Type"
        ],
        group_by_list = [
          "Inventory.Type"
        ]
      }
      field = {
        name = "Inventory.Type",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "cisa_kev_public_assets_widget" {
  name               = "Known Exploited Vulnerabilities On Public Facing Assets"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "Known Exploited Vulnerabilities On Public Facing Assets",
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
                  "Vulnerabilities"
                ],
                "operator" : "in"
              },
              {
                "key" : "Status",
                "type" : "str",
                "values" : [
                  "open",
                  "in_progress"
                ],
                "operator" : "in"
              },
              {
                "key" : "Labels",
                "type" : "list",
                "values" : [
                  {
                    "key" : "Labels",
                    "type" : "str",
                    "values" : [
                      "cisa_kev"
                    ],
                    "operator" : "in"
                  }
                ],
                "operator" : "any_match"
              },
              {
                "keys" : [
                  "Inventory"
                ],
                "type" : "object_set",
                "with" : {
                  "key" : "Exposure",
                  "type" : "str",
                  "values" : [
                    "public_facing"
                  ],
                  "operator" : "in"
                },
                "models" : [
                  "Inventory"
                ],
                "operator" : "has"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Alert"
          ]
        })
        group_by : [
          "Inventory.Type"
        ],
        group_by_list = [
          "Inventory.Type"
        ]
      }
      field = {
        name = "Inventory.Type",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "public_data_widget" {
  name               = "Data Storage and Databases Publicly Exposed"
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
          "keys" : [
            "Inventory"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "key" : "NewCategory",
                "type" : "str",
                "values" : [
                  "Data Storage"
                ],
                "operator" : "in"
              },
              {
                "key" : "Exposure",
                "type" : "str",
                "values" : [
                  "public_facing"
                ],
                "operator" : "in"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Inventory"
          ]
        })
        group_by : [
          "Type"
        ],
        group_by_list = [
          "Type"
        ]
      }
      field = {
        name = "Type",
        type = "str"
      }
    }
  }
}


output "public_facing_assets_widget_id" {
  value = orcasecurity_custom_widget.public_facing_assets_widget.id
}

output "vm_public_ingress_widget_id" {
  value = orcasecurity_custom_widget.vm_public_ingress_widget.id
}

output "public_facing_neglected_compute_widget_id" {
  value = orcasecurity_custom_widget.public_facing_neglected_compute_widget.id
}

output "attack_paths_public_widget_id" {
  value = orcasecurity_custom_widget.attack_paths_public_widget.id
}

output "public_assets_priority_risk_widget_id" {
  value = orcasecurity_custom_widget.public_facing_assets_widget.id
}

output "cisa_kev_public_assets_widget_id" {
  value = orcasecurity_custom_widget.cisa_kev_public_assets_widget.id
}

output "public_data_widget_id" {
  value = orcasecurity_custom_widget.public_data_widget.id
}


