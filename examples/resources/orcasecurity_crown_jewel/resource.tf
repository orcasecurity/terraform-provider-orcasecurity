# description is the same field as Reason in the Orca UI ("Mark as Crown Jewel").
# Typical UI values: "Critical business function", "Customer data", "High blast radius",
# or free text when choosing Other.
#
# Apply is an upsert: if this asset is already user-marked, the existing Reason is
# overwritten. To adopt an existing mark without changing it, import first.
resource "orcasecurity_crown_jewel" "example" {
  group_unique_id = "vm_123456789012_i-0123456789abcdef0"
  description     = "Customer data"
}
