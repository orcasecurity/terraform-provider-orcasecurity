# Changelog

Notable changes in each release of the Orca Security Terraform provider. The same notes are published on [GitHub Releases](https://github.com/orcasecurity/terraform-provider-orcasecurity/releases).

## 0.0.62 (October 7, 2026)

### Changes

* WASP-1642: Manage business units through /api/business_units ([#277](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/277))

## 0.0.61 (September 24, 2026)

### Changes

* WASP-1702: Survive API throttling on large custom alert configs ([#275](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/275))

## 0.0.60 (September 22, 2026)

### Changes

* WASP-1672: stop Terraform wiping custom compliance framework controls ([#274](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/274))

### Dependencies

* Chore(deps): bump google.golang.org/grpc from 1.83.1 to 1.83.2 ([#273](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/273))

## 0.0.59 (September 8, 2026)

### Changes

* WASP-1651: Use Orca tenant partition in S3 bucket policy ARNs ([#272](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/272))

### Dependencies

* Chore(deps): bump google.golang.org/grpc from 1.82.1 to 1.83.1 ([#270](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/270))

## 0.0.58 (September 2, 2026)

### Changes

* WASP-1631: Add compliance framework selection, data sources, and nested custom-framework sections ([#269](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/269))
* WASP-1553: Add user-defined crown jewel resource and data source ([#268](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/268))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.48.0 to 0.49.0 ([#267](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/267))
* Bump github.com/hashicorp/terraform-plugin-log from 0.10.0 to 0.11.0 ([#265](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/265))

## 0.0.57 (July 31, 2026)

### Bug fixes

* fix(tests): stabilize lab acceptance tests and provider drift (WASP-1497) ([#263](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/263))
* fix(rbac-access): stop group/user access re-creation and silent-delete ([#261](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/261)) ([#262](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/262))

### Dependencies

* Bump google.golang.org/grpc from 1.79.3 to 1.82.1 ([#260](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/260))

## 0.0.56 (July 20, 2026)

### Features

* feat(automation): remove deprecated orcasecurity_automation (v1) ([#257](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/257))

### Changes

* WASP-1459: automation V2 priority management (attribute, data source, bulk ordering resource) ([#256](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/256))
* Fix data detection rule priority race + add tests for 18 untested packages ([#254](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/254))
* WASP-1444\Add support DSPM Data Detection configuration for terraform provider ([#251](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/251))
* WASP-1446\Add admission controller resources and data source ([#252](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/252))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.47.0 to 0.48.0 ([#253](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/253))
* Bump actions/setup-go from 6 to 7 ([#255](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/255))

## 0.0.55 (July 12, 2026)

### Changes

* Wasp 1441 ([#250](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/250))
* WASP-1406/add integration resources for external services ([#245](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/245))

### Dependencies

* Bump golang.org/x/crypto from 0.51.0 to 0.52.0 ([#249](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/249))
* Bump golang.org/x/net from 0.53.0 to 0.55.0 ([#246](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/246))

## 0.0.54 and earlier

See [GitHub Releases](https://github.com/orcasecurity/terraform-provider-orcasecurity/releases).
