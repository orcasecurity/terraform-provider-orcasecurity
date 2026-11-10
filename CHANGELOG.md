# Changelog

Release notes for 0.0.63 and later are on [GitHub Releases](https://github.com/orcasecurity/terraform-provider-orcasecurity/releases). This file keeps the notes for earlier releases.

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

* WASP-1553: Add user-defined crown jewel resource and data source ([#268](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/268))
* WASP-1631: Add compliance framework selection, data sources, and nested custom-framework sections ([#269](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/269))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.48.0 to 0.49.0 ([#267](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/267))
* Bump github.com/hashicorp/terraform-plugin-log from 0.10.0 to 0.11.0 ([#265](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/265))

## 0.0.57 (July 31, 2026)

### Bug fixes

* fix(rbac-access): stop group/user access re-creation and silent-delete ([#261](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/261)) ([#262](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/262))
* fix(tests): stabilize lab acceptance tests and provider drift (WASP-1497) ([#263](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/263))

### Dependencies

* Bump google.golang.org/grpc from 1.79.3 to 1.82.1 ([#260](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/260))

## 0.0.56 (July 20, 2026)

### Features

* feat(automation): remove deprecated orcasecurity_automation (v1) ([#257](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/257))

### Bug fixes

* Fix data detection rule priority race + add tests for 18 untested packages ([#254](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/254))

### Changes

* WASP-1444\Add support DSPM Data Detection configuration for terraform provider ([#251](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/251))
* WASP-1446\Add admission controller resources and data source ([#252](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/252))
* WASP-1459: automation V2 priority management (attribute, data source, bulk ordering resource) ([#256](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/256))

### Dependencies

* Bump actions/setup-go from 6 to 7 ([#255](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/255))
* Bump github.com/getsentry/sentry-go from 0.47.0 to 0.48.0 ([#253](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/253))

## 0.0.55 (July 12, 2026)

### Changes

* WASP-1406/add integration resources for external services ([#245](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/245))
* Wasp 1441 ([#250](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/250))

### Dependencies

* Bump golang.org/x/crypto from 0.51.0 to 0.52.0 ([#249](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/249))
* Bump golang.org/x/net from 0.53.0 to 0.55.0 ([#246](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/246))

## 0.0.54 (June 29, 2026)

### Changes

* WASP-1399/fix widget gaps ([#243](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/243))
* WASP-1401/allow empty reason and justification in automation_v2 ([#244](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/244))
* add group_by_2 to supprot sort inside group ([#242](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/242))

## 0.0.53 (June 23, 2026)

### Changes

* WASP-1377/the resource names do not follow the recommendation of using underscore instead of hyphens ([#241](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/241))

### Dependencies

* Bump actions/checkout from 6 to 7 ([#240](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/240))

## 0.0.52 (June 18, 2026)

### Changes

* ST-980\fix bug on delete UI for discovery view ([#238](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/238))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.46.2 to 0.47.0 ([#237](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/237))

## 0.0.51 (June 15, 2026)

Same code as 0.0.50; no changes.

## 0.0.50 (June 15, 2026)

### Changes

* WASP-1358\add columns sort groupBy to orcasecurity_discovery_view ([#236](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/236))

## 0.0.49 (June 11, 2026)

### Changes

* WASP-1345\adding support to customer tag rules ([#234](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/234))
* WASP-1346\add support for Scheduled Reports  ([#235](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/235))

## 0.0.48 (June 9, 2026)

### Changes

* WASP-1329\Add AppSec APIs in orca terraform provider ([#233](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/233))

## 0.0.47 (May 21, 2026)

### Changes

* WASP-1262\(automation_v2): add apply_on_existing flag ([#232](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/232))

## 0.0.46 (May 20, 2026)

Same code as 0.0.45; no changes.

## 0.0.45 (May 20, 2026)

### Changes

* docs: align unit test command and example outputs with Terraform CLI ([#227](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/227))
* remove legacy sonar query data source (WASP-1187) ([#231](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/231))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.46.0 to 0.46.2 ([#229](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/229))

## 0.0.44 (April 28, 2026)

### Changes

* ci: use golangci-lint v2 for Go 1.25 and bump lint action ([#226](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/226))

### Dependencies

* Bump github.com/getsentry/sentry-go from 0.45.1 to 0.46.0 ([#224](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/224))
* Bump github.com/hashicorp/terraform-plugin-docs from 0.24.0 to 0.25.0 ([#223](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/223))
* Bump github.com/hashicorp/terraform-plugin-testing from 1.13.3 to 1.16.0 ([#222](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/222))

## 0.0.43 (April 28, 2026)

### Changes

* WASP-1126\add retry HTTP error 5xx on terraform provider ([#219](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/219))

## 0.0.42 (April 14, 2026)

### Features

* feat: add support for custom compliance frameworks creationg and mana… ([#217](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/217))

### Changes

* WASP-1120\add permission in Groups ([#218](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/218))

### Dependencies

* Bump crazy-max/ghaction-import-gpg from 5 to 7 ([#214](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/214))
* Bump github.com/cloudflare/circl from 1.6.1 to 1.6.3 ([#212](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/212))
* Bump github.com/getsentry/sentry-go from 0.35.3 to 0.40.0 ([#193](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/193))
* Bump github.com/hashicorp/terraform-plugin-framework-validators ([#174](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/174))
* Bump github.com/hashicorp/terraform-plugin-log from 0.9.0 to 0.10.0 ([#181](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/181))
* Bump google.golang.org/grpc from 1.75.1 to 1.79.3 ([#216](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/216))
* Bump goreleaser/goreleaser-action from 6 to 7 ([#210](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/210))
* Bump hashicorp/setup-terraform from 3 to 4 ([#211](https://github.com/orcasecurity/terraform-provider-orcasecurity/pull/211))

## 0.0.41 and earlier

See [GitHub Releases](https://github.com/orcasecurity/terraform-provider-orcasecurity/releases).
