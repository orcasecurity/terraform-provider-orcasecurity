Developing the Provider
---------------------------

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (version 1.18+ is *required*). You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

To compile the provider, run `make install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make install
...
$ $GOPATH/bin/terraform-provider-orcasecurity
...
```

In order to test the provider, you can simply run `make test-unit`.

```sh
$ make test-unit
```

In order to run the full suite of acceptance tests, run `make test-acc`.

*Note:* Acceptance tests create real resources. You have to be a Orca Security customer to execute them.

Export provider configuration:
```sh
export ORCASECURITY_API_ENDPOINT=https://api.orcasecurity.io # US region
export ORCASECURITY_API_TOKEN=YOURSECRETAPITOKEN
```

```sh
$ make test-acc
```

In order to run a specific acceptance test, use the `TESTARGS` environment variable. For example, the following command will run `TestAccAutomationResource_JiraIssue` acceptance test only:

```sh
$ make test-acc TESTARGS='-run=TestAccAutomationResource_JiraIssue'
```

All acceptance tests for a specific package can be run by setting the `PKG_NAME` environment variable. For example:

```sh
$ make test-acc PKG_NAME=orcasecurity/automations
```

In order to check changes you made locally to the provider, you can use the binary you just compiled by adding the following
to your `~/.terraformrc` file. This is valid for Terraform 1.0+. Please see
[Terraform's documentation](https://www.terraform.io/docs/cli/config/config-file.html#development-overrides-for-provider-developers)
for more details.

```
provider_installation {

  # Use /home/yourname/go/bin as an overridden package directory
  # for the orcasecurity/orcasecurity provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # orcasecurity provider plugin in the given directory.
  dev_overrides {
    "orcasecurity/orcasecurity" = "/home/yourname/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

For information about writing acceptance tests, see the main Terraform [contributing guide](https://github.com/hashicorp/terraform/blob/master/.github/CONTRIBUTING.md#writing-acceptance-tests).

Releasing
---------------------------

Every merged PR is released automatically. Its release notes are the `CHANGELOG.md` section that the PR itself adds, and the same text is published as the GitHub release, which is where Dependabot and similar tools read it.

1. Open your PR (a draft is fine), then run `make changelog` on the branch. It adds a `## X.Y.Z (date)` section to the top of `CHANGELOG.md` for the version your merge will release. The section lists your PR (from its title) and anything merged since the last release, such as Dependabot bumps, grouped into Features, Bug fixes, Changes and Dependencies (rules in `.goreleaser.yml` under `changelog`). The first run builds a pinned GoReleaser into `bin/`.
2. Reword entries if you like, then commit `CHANGELOG.md` and push. The **CHANGELOG.md** check fails the PR if the section is missing, has the wrong version, or leaves out your PR or an unreleased change.
3. When the PR merges, the **Release on merge** workflow tags the merge commit and publishes the release with that section as its notes.

Labels change what a merge releases (rerun `make changelog` after changing them):

| Label | Effect |
| --- | --- |
| *(none)* | patch release, e.g. 0.0.63 → 0.0.64 |
| `release:minor` | minor release, e.g. 0.0.63 → 0.1.0 |
| `release:major` | major release, e.g. 0.1.0 → 1.0.0 |
| `no-release` | no section, no release; the change ships with the next release |

If another PR is released first, your branch conflicts in `CHANGELOG.md`: update the branch from `master` and rerun `make changelog`. Rerunning rebuilds the section from scratch, so reword entries after the last run. Dependabot PRs need no section; their bumps are listed in the next release.

Pushing a `v*` tag by hand still releases, with notes generated from the commits; the next `make changelog` backfills that version into `CHANGELOG.md`.
