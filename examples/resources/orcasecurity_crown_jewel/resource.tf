# description is the same field as Reason in the Orca UI ("Mark as Crown Jewel").
# Typical UI values: "Critical business function", "Customer data", "High blast radius",
# or free text when choosing Other.
#
# Create fails if this asset is already user-marked or Orca-detected (same as the UI).
# Import first to adopt an existing user mark. group_unique_id must exist in inventory.
resource "orcasecurity_crown_jewel" "example" {
  group_unique_id = "vm_123456789012_i-0123456789abcdef0"
  description     = "Customer data"

  timeouts = {
    create = "90s"
    update = "90s"
    delete = "90s"
  }
}
