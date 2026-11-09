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

Release notes live in [CHANGELOG.md](./CHANGELOG.md) and are copied into the GitHub release, which is where Dependabot and similar tools read them.

Releases are driven by the **Release PR** workflow (`.github/workflows/release-pr.yml`):

1. On every push to `master`, it opens or refreshes a PR titled `Release vX.Y.Z` (next patch version) whose only change is the new `CHANGELOG.md` section, built from the commits since the last tag and grouped into Features, Bug fixes, Changes and Dependencies (rules in `.goreleaser.yml` under `changelog`).
2. To release, approve and squash-merge that PR. The workflow tags the merge commit and runs `release.yml`, which publishes the `CHANGELOG.md` section as the release notes.
3. For a non-patch version, run the workflow manually with the `version` input (e.g. `0.1.0`); the open release PR is retitled.

To reword an entry, edit `CHANGELOG.md` on the `release/next` branch right before merging; the workflow rebuilds the branch whenever `master` changes. To preview the next section locally, run `make release-notes VERSION=X.Y.Z` (do not commit the result).

Pushing a `v*` tag by hand still releases, with notes generated from the commits; the next release PR then backfills that version into `CHANGELOG.md`.
