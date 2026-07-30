### Shift Left (AppSec) acceptance tests

The Shift Left SCM resources adopt objects that already exist in the tenant: an account, workspace or
group is integrated through the SCM's own install flow, and the provider then manages its
configuration. There is nothing for a test to create from scratch, so these tests need a tenant that
already has an installation and at least one unit under it, identified by environment variables. Each
test skips with the names it wanted when they are unset, so an incomplete environment is quiet rather
than red.

| Variable | Points at |
|---|---|
| `ORCA_TEST_GH_ACCOUNT_ID` | Orca id of an integrated GitHub account |
| `ORCA_TEST_GL_INSTALLATION_ID` | GitLab installation holding the group below |
| `ORCA_TEST_GL_GITLAB_GROUP_ID` | Numeric GitLab group id (or `ORCA_TEST_GL_GROUP_ID` for the Orca id) |
| `ORCA_TEST_GL_PROJECT_ID` | Shift Left project the group may be moved into |
| `ORCA_TEST_BB_INSTALLATION_ID` | Bitbucket installation holding the workspace below |
| `ORCA_TEST_BB_ACCOUNT_SLUG` | Bitbucket workspace slug (or `ORCA_TEST_BB_ORCA_ACCOUNT_ID` for the Orca id) |
| `ORCA_TEST_AZ_INSTALLATION_ID` | Azure DevOps installation holding the organization below |
| `ORCA_TEST_AZ_ACCOUNT_NAME` | Azure DevOps organization name (or `ORCA_TEST_AZ_ACCOUNT_ID` for the Orca id) |
| `ORCA_TEST_PROJECT_ID` | Shift Left project a policy may be attached to |
| `ORCA_TEST_BUILTIN_POLICY_TYPE` / `ORCA_TEST_BUILTIN_POLICY_ID` | A built-in `licenses` policy to attach projects to |
| `ORCA_TEST_MALICIOUS_PACKAGES_POLICY_ID` | The built-in `malicious_packages` policy |
| `ORCASECURITY_ACC_SCM_INSTALLATION_ID` | GitHub installation used by the SCM posture policy test |

Terraform destroys everything it created when a test case ends, and for an adopted unit that means
de-integrating a unit the test did not integrate — which also drops every repository under it, and the
restore only puts the empty unit back. Those cases therefore need a second opt-in
(`ORCA_TEST_GH_ALLOW_DESTROY`, `ORCA_TEST_GL_ALLOW_DESTROY`, `ORCA_TEST_BB_ALLOW_DESTROY`,
`ORCA_TEST_AZ_ALLOW_DESTROY`) and skip themselves when the unit they were pointed at has repositories.
Point them at a disposable empty unit, never at a shared one.

Two areas are deliberately not covered against a live tenant:

- **Installation resources** (GitLab, Bitbucket, Azure DevOps). Creating one requires a working personal
  access token for the SCM itself, which is a credential the test environment cannot be expected to
  hold. Their request and response mapping is unit-tested instead (`mapping_test.go` in each
  installation package). GitHub has no installation resource at all: that installation is created by
  the GitHub App flow, outside of Terraform.
- **Repository create/update/delete.** Integrating a repository and then destroying it deletes its
  repository context in the tenant. `shift_left_repository/import_apply_test.go` drives import, apply,
  update, replace and destroy against a stateful in-process stub of the API instead, so the lifecycle is
  covered without a tenant.
