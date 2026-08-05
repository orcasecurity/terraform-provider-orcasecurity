### Shift Left (AppSec) acceptance tests

Shift Left SCM acceptance tests adopt tenant objects already integrated via each SCM's install flow — they cannot create units from scratch. Tests need env vars pointing at an existing installation and unit; unset vars cause a quiet skip, not a failure.

#### Identity cheat-sheet (`account_id` and friends)

| Attribute | Resource / DS | Meaning |
|---|---|---|
| `account_id` | GitHub account / repository | **Orca UUID** of the GitHub App installation unit |
| `account_id` | Bitbucket account / repository | **Bitbucket slug** (workspace/project key), not an Orca UUID |
| `account_id` | Bitbucket installation | Token-scope workspace/project slug on the credential |
| `id` | Bitbucket account | Orca unit UUID (use this when you need the Orca id) |
| `installation_id` | GitLab / Bitbucket / Azure unit or repo | Orca parent installation UUID |

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
| `ORCASECURITY_ACC_SCM_INSTALLATION_ID` | GitHub installation used by the SCM posture policy test (must be a live install the API can assign scope to; deleted/orphan ids make create fail) |
| `ORCA_TEST_SCM_POSTURE_DEFAULT_ALLOW` | Opt-in for the org-wide SCM posture default-policy adopt test (`orcasecurity_shift_left_scm_posture_default_policy`). Unset = skip. The test snapshots and restores the singleton; without this gate a normal `TF_ACC` suite would mutate every org's built-in posture policy. |

Destroy on adopted units de-integrates them and drops all repos; restore only recreates the empty unit. Opt in with `ORCA_TEST_GH_ALLOW_DESTROY`, `ORCA_TEST_GL_ALLOW_DESTROY`, `ORCA_TEST_BB_ALLOW_DESTROY`, or `ORCA_TEST_AZ_ALLOW_DESTROY`; tests skip when repos exist. Use a disposable empty unit — GitHub destroy has no Integrate restore and needs the App flow to recover.

Two areas are not covered against a live tenant:

- **Installation resources** (GitLab, Bitbucket, Azure DevOps). Creating one needs a real SCM PAT the test env cannot hold; mapping is unit-tested in `orcasecurity/shift_left_installation` instead. GitHub has no installation resource (App flow only). Installation-list and repository-list data sources are GET-only and do have live tests under plain `TF_ACC`.
- **Repository create/update/delete.** Destroy deletes repository context in the tenant. `shift_left_repository/import_apply_test.go` covers import, apply, update, replace, and destroy against an in-process API stub instead.
