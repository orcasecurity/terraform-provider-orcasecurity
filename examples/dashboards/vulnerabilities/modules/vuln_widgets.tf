terraform {
  required_providers {
    orcasecurity = {
      source = "orcasecurity/orcasecurity"
    }
  }
}

resource "orcasecurity_custom_widget" "vulnerability_alerts_widget" {
  name               = "Vulnerabilities - Risk Prioritized Alerts"
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
          "models" : [
            "Alert"
          ],
          "type" : "object_set",
          "with" : {
            "operator" : "and",
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
                "key" : "Category",
                "type" : "str",
                "values" : [
                  "Vulnerabilities"
                ],
                "operator" : "in"
              },
              {
                "keys" : [
                  "CloudAccount"
                ],
                "type" : "object",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "key" : "CloudProvider",
                      "type" : "str",
                      "values" : [
                        "shiftleft"
                      ],
                      "operator" : "not_eq"
                    }
                  ],
                  "operator" : "and"
                },
                "model" : "CloudAccount",
                "operator" : "has"
              }
            ],
        } })
        group_by : [
          "Name"
        ]
        start_at_index    = 0
        order_by          = ["Score"]
        limit             = 10
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "priority_vulnerabilities_alerts_widget" {
  name               = "Critical & High Vulnerability Alerts"
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
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "models" : [
            "Alert"
          ],
          "type" : "object_set",
          "with" : {
            "operator" : "and",
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
                "keys" : [
                  "CloudAccount"
                ],
                "type" : "object",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "key" : "CloudProvider",
                      "type" : "str",
                      "values" : [
                        "shiftleft"
                      ],
                      "operator" : "not_eq"
                    }
                  ],
                  "operator" : "and"
                },
                "model" : "CloudAccount",
                "operator" : "has"
              },
              {
                "key" : "CreatedAt",
                "type" : "datetime",
                "values" : [
                  14
                ],
                "operator" : "before_past",
                value_type : "days"
              },
              {
                "key" : "RiskLevel",
                "type" : "str",
                "values" : [
                  "critical",
                  "high"
                ],
                "operator" : "in"
              },
            ],
        } })
        group_by : [
          "Name"
        ]
        start_at_index    = 0
        order_by          = ["Score"]
        limit             = 100
        enable_pagination = true
      }
    }
  }
}

resource "orcasecurity_custom_widget" "cisa_kev_public_assets_widget" {
  name               = "CISA KEV Vulnerabilities On Public Facing Assets"
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
          "models" : [
            "Alert"
          ],
          "type" : "object_set",
          "with" : {
            "operator" : "and",
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
        } })
        group_by : [
          "Name"
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

resource "orcasecurity_custom_widget" "fixable_critical_vulnerabilities_widget" {
  name               = "Critical Vulnerabilities with Fixes Available"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "Discovered Over 14 Days Ago",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "keys" : [
            "Vulnerability"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "keys" : [
                  "CVE"
                ],
                "type" : "object_set",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "keys" : [
                        "VulnerablePackages"
                      ],
                      "type" : "object_set",
                      "with" : {
                        "key" : "Fixed",
                        "type" : "bool",
                        "values" : [
                          true
                        ],
                        "operator" : "eq"
                      },
                      "models" : [
                        "VulnerablePackage"
                      ],
                      "operator" : "has"
                    },
                    {
                      "key" : "Cvss3Severity",
                      "type" : "str",
                      "values" : [
                        "CRITICAL"
                      ],
                      "operator" : "in"
                    },
                    {
                      "key" : "HasExploit",
                      "type" : "bool",
                      "values" : [
                        true
                      ],
                      "operator" : "eq"
                    }
                  ],
                  "operator" : "and"
                },
                "models" : [
                  "CVE"
                ],
                "operator" : "has"
              },
              {
                "keys" : [
                  "CVEVendorData"
                ],
                "type" : "object_set",
                "models" : [
                  "CVEDescription"
                ],
                "operator" : "has"
              },
              {
                "key" : "FirstSeen",
                "type" : "datetime",
                "values" : [
                  14
                ],
                "operator" : "before_past",
                "value_type" : "days"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Vulnerability"
          ]
        })
        group_by : [
          "Name"
        ],
        group_by_list = [
          "Name"
        ]
      }
      field = {
        name = "Name",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "priority_vulnerabilities_alerts_source_widget" {
  name               = "Vulnerabilities - Critical and High Alerts by Source"
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
            "Vulnerability"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "keys" : [
                  "CVE"
                ],
                "type" : "object_set",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "keys" : [
                        "VulnerablePackages"
                      ],
                      "type" : "object_set",
                      "models" : [
                        "VulnerablePackage"
                      ],
                      "operator" : "has"
                    },
                    {
                      "key" : "Cvss3Severity",
                      "type" : "str",
                      "values" : [
                        "CRITICAL"
                      ],
                      "operator" : "in"
                    }
                  ],
                  "operator" : "and"
                },
                "models" : [
                  "CVE"
                ],
                "operator" : "has"
              },
              {
                "keys" : [
                  "CVEVendorData"
                ],
                "type" : "object_set",
                "models" : [
                  "CVEDescription"
                ],
                "operator" : "has"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Vulnerability"
          ]
        })
        group_by : [
          "Name"
        ],
        group_by_list = [
          "Name"
        ]
      }
      field = {
        name = "Name",
        type = "str"
      }
    }
  }
}

resource "orcasecurity_custom_widget" "prevalent_critical_vulnerabilities_widget" {
  name               = "Most Prevalent Critical Vulnerabilities"
  organization_level = true
  extra_params = {
    type                = "donut",
    empty_state_message = "No data found",
    default_size        = "sm",
    is_new              = true,
    subtitle            = "Grouped by CVE",
    description         = "",
    settings = {
      columns = [
        "alert",
        "asset"
      ]
      request_params = {
        query = jsonencode({
          "keys" : [
            "Vulnerability"
          ],
          "type" : "object_set",
          "with" : {
            "type" : "operation",
            "values" : [
              {
                "keys" : [
                  "CVE"
                ],
                "type" : "object_set",
                "with" : {
                  "type" : "operation",
                  "values" : [
                    {
                      "keys" : [
                        "VulnerablePackages"
                      ],
                      "type" : "object_set",
                      "models" : [
                        "VulnerablePackage"
                      ],
                      "operator" : "has"
                    },
                    {
                      "key" : "Cvss3Severity",
                      "type" : "str",
                      "values" : [
                        "CRITICAL"
                      ],
                      "operator" : "in"
                    }
                  ],
                  "operator" : "and"
                },
                "models" : [
                  "CVE"
                ],
                "operator" : "has"
              },
              {
                "keys" : [
                  "CVEVendorData"
                ],
                "type" : "object_set",
                "models" : [
                  "CVEDescription"
                ],
                "operator" : "has"
              }
            ],
            "operator" : "and"
          },
          "models" : [
            "Vulnerability"
          ]
        })
        group_by : [
          "Name"
        ],
        group_by_list = [
          "Name"
        ]
      }
      field = {
        name = "Name",
        type = "str"
      }
    }
  }
}

output "vulnerability_alerts_widget_id" {
  value = orcasecurity_custom_widget.vulnerability_alerts_widget.id
}

output "priority_vulnerabilities_alerts_widget_id" {
  value = orcasecurity_custom_widget.priority_vulnerabilities_alerts_widget.id
}

output "cisa_kev_public_assets_widget_id" {
  value = orcasecurity_custom_widget.cisa_kev_public_assets_widget.id
}

output "fixable_critical_vulnerabilities_widget_id" {
  value = orcasecurity_custom_widget.fixable_critical_vulnerabilities_widget.id
}

output "priority_vulnerabilities_alerts_source_widget_id" {
  value = orcasecurity_custom_widget.priority_vulnerabilities_alerts_source_widget.id
}

output "prevalent_critical_vulnerabilities_widget_id" {
  value = orcasecurity_custom_widget.prevalent_critical_vulnerabilities_widget.id
}