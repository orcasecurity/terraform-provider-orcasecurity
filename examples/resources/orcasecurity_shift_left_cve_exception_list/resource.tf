//Shift Left CVE Exception List
resource "orcasecurity_shift_left_cve_exception_list" "shiftleft_exception_list_1" {
  name        = "Exception List with Terraform"
  description = "Log4Shell Exception List"
  disabled    = false
  vulnerabilities = [
    {
      cve_id      = "cve-2021-44228"
      description = "log4shell"
      disabled    = false
      expiration  = "2024/09/25"
    }
  ]
}