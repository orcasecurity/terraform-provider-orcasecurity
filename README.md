# Terraform Provider for Orca Security

TODO:
- clone the repo
- describe directory layout
- add .terraformrc
- running terraform locally
- how to generate docs
- how to add documentation examples
- add .envrc or export vars
- how to run unit tests
- how to run acceptance tests
- how to publish to registry
- gpg setup for publishing
- add links to official docs and tutorials


# publishing
## gpg key generation
https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key

- gpg --full-generate-key  # at least 4096
- gpg --list-secret-keys --keyid-format=long
- gpg --armor --export 3AA5C34371567BD2
  