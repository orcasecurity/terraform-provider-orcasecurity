variable "group_id" {
  description = "The ID of the group."
  type        = string
}

variable "role_id" {
  description = "The ID of the role."
  type        = string
}

variable "all_cloud_accounts" {
  description = "Whether the group has access to all cloud accounts."
  type        = bool
}

variable "cloud_accounts" {
  description = "List of cloud account IDs."
  type        = list(string)
  default     = []
}

variable "business_units" {
  description = "List of business unit IDs."
  type        = list(string)
  default     = []
}

variable "shiftleft_projects" {
  description = "List of Shiftleft project IDs."
  type        = list(string)
  default     = []
}
