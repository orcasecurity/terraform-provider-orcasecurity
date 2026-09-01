# description is the same field as Reason in the Orca UI ("Mark as Crown Jewel").
# UI presets: "Data: Personal identifiable information", "Access: Broad permission access",
# "Access: Secrets exposure", "Data: Intellectual property", "Data: Financial information",
# or Other (free text, max 50 characters).
#
# Create fails if this asset is already user-marked. Import first to adopt an
# existing mark. group_unique_id must exist in inventory. Orca-detected assets
# can still be marked.
resource "orcasecurity_crown_jewel" "example" {
  group_unique_id = "vm_123456789012_i-0123456789abcdef0"
  description     = "Data: Financial information"

  timeouts = {
    create = "90s"
    update = "90s"
    delete = "90s"
  }
}
