# Changelog

## 0.11.0

`0.11.0` is a large provider-coverage and Terraform compatibility release. It
adds broad import support across AWS, Kubernetes, Datadog, Cloudflare, and
LaunchDarkly, moves generated state onto Terraform 1.x-compatible provider
source addresses, and hardens provider discovery so failures surface as returned
errors instead of silent skips or process exits.

## What's Changed

### Compatibility And Release Notes

* Add a Terraform 1.x state compatibility bridge, emit typed state attributes,
  and document support for Terraform CLI 1.9 through 1.14.
* Decouple Terraform compatibility code into `terraformutils/tfcompat` and add
  scheduled compatibility checks for Terraform state and provider registry
  behavior.
* Adopt `v`-prefixed release tags starting with `v0.11.0` so Go module version
  discovery works correctly. Earlier releases used plain tags such as `0.10.0`.
* Update MyraSec to the current provider client line and remove the obsolete
  `myrasec_ratelimit` resource from the documented support list.

### Provider Import Coverage

* Expand AWS coverage across API Gateway v2, AppConfig, App Mesh, AppSync,
  Backup, CloudFront, CloudWatch Logs/EventBridge, Cognito, Config, DynamoDB,
  ECR, ECS/Lambda, EKS, ElastiCache, Glue, IAM federation/account settings,
  Kinesis, MSK, RDS, S3 bucket configuration, Secrets Manager, SQS, and SSM.
* Expand Kubernetes coverage with modern typed resources, stable `v1` mappings,
  default service accounts, labels, node taints, ConfigMap and Secret data,
  workload environment variables, CRDs, and manifest-backed native API support
  for the Kubernetes 1.33 through 1.35 support window.
* Expand Datadog coverage with cloud inventory sync config, monitor JSON and
  notification/config policies, metric tag configuration, SLO corrections, spans
  and RUM metrics, RUM applications/retention filters, security monitoring
  filters and suppressions, sensitive data scanner resources, team resources,
  and On-Call resources.
* Expand Cloudflare coverage for Access/Zero Trust, certificates, DNS records,
  Email Routing, lists, load balancing, Logpush, Magic WAN, notifications,
  Pages, rulesets, storage resources, Turnstile, tunnels, waiting rooms, web
  analytics, and Workers resources.
* Expand LaunchDarkly coverage from project-only support to standalone
  environments, access tokens, AI/model resources, custom roles, destinations,
  feature-flag-related resources, metrics, relay proxy configuration, segments,
  teams, team members, views, view links, webhooks, and integrations.

### Discovery And Error Handling

* Return discovery/list/read errors from AWS, GCP, GitHub, GitLab, Okta, Logz.io,
  Keycloak, Azure, IBM, RabbitMQ, MyraSec, Datadog, Auth0, Opal, and shared
  provider paths instead of dropping failures during import discovery.
* Harden provider initialization by validating required and optional init args,
  clearing stale init/service state before retry and selection paths, staging
  optional state only after success, and surfacing ignored parse errors.
* Stop AWS pagination loops on empty continuation tokens and propagate nested
  AWS discovery errors for ECS, EFS children, and Organizations.
* Close Datadog and LaunchDarkly API response bodies and include stable IDs in
  LaunchDarkly generated resource names.
* Refactor shared provider setup and command generator registry code so provider
  metadata, service lookup, and selection behavior stay consistent.

### Dependencies, CI, And Repository Maintenance

* Refresh a large dependency set, including AWS SDK v2 service modules,
  Cloudflare, Datadog, LaunchDarkly, Auth0, GitLab, Azure helpers, Honeycomb,
  IBM SDKs, OctopusDeploy, MyraSec, CommerceTools, Alicloud TableStore, Okta,
  TencentCloud, Google APIs, and shared HashiCorp modules.
* Remove unused or incompatible dependency paths, including the old Terraform
  module dependency and several direct `+incompatible` module edges.
* Add command, terraformutils, provider name/ID extraction, low-coverage package,
  state compatibility, and provider registry compatibility tests.
* Clear the legacy lint baseline, suppress test-only gosec noise, add changed
  line lint behavior, cancel stale PR workflow runs, add provider PR labeling,
  and wire the release workflow to GoReleaser with draft releases.
* Replace legacy license boilerplate with SPDX headers and document the MPL-2.0
  boundary for Terraform compatibility code.

### Follow-Up Tracking

* Continue AWS provider parity work in #338. The current tracking snapshot has
  Terraformer importing about 407 AWS resource types against roughly 1,645
  Terraform AWS provider resources, with the remaining work split into
  practical service-family PRs.
* Continue Kubernetes close-out work in #337, including the Kubernetes 1.33
  through 1.35 native manifest policy, provider schema audit, intentionally
  unsupported resource documentation, and matrix fixtures for expected discovery
  behavior.
* Continue Datadog coverage work in #336. After the recent expansion, Terraformer
  has 61 registered Datadog services against about 130 upstream resources, with
  the remaining high-confidence importable gap tracked as follow-up waves.
* Continue Cloudflare coverage work in #335. The tracker separates useful
  importable resources from Cloudflare-managed, singleton, request-style, or
  write-only resources so parity work does not produce unusable generated HCL.

**Full Changelog**: https://github.com/chenrui333/terraformer/compare/0.10.0...v0.11.0

## 0.10.0

`0.10.0` is a minor maintenance release that continues Terraformer's post-fork
dependency recovery. It clears the known module-level vulnerability findings,
expands changed-line lint coverage, refreshes a large set of provider SDKs, and
keeps the remaining source-heavy provider and Terraform-core migrations tracked
separately for follow-up work.

## What's Changed

### Security And CI

* Update Okta SDK dependencies and replace the legacy `jwt-go` module path so
  `govulncheck` module scans are clean.
* Expand golangci-lint changed-line audit coverage.
* Let PR workflows run for branch-to-branch stacked PRs during large backlog
  triage.

### Dependency And Provider Maintenance

* Refresh Datadog, Okta, GitHub, Opsgenie, Fastly, Equinix Metal, PAN-OS, IBM,
  Azure, Keycloak, and shared HashiCorp/ZCL dependencies.
* Batch-update provider client libraries across Azure, IONOS, Mackerel,
  DigitalOcean, Linode, OpenStack, TencentCloud, Alicloud, Yandex, Grafana,
  PagerDuty, Mikrotik, Opal, NS1, and Logz.io.
* Migrate the Logz.io provider to the maintained Terraform client module.
* Migrate the Keycloak provider import path to the maintained module path.

### Follow-Up Tracking

* Track the remaining broad SDK and Terraform-core migrations in issue #155
  instead of blocking this release on source-heavy provider rewrites.

**Full Changelog**: https://github.com/chenrui333/terraformer/compare/0.9.0...0.10.0

## 0.9.0

`0.9.0` is a minor maintenance release that refreshes Terraformer's supported
toolchain and dependency base after the repo resumed active maintenance. It raises
the Go floor to 1.26.2, modernizes GitHub Actions and CI, restores dependency
automation with Renovate, and updates core provider dependencies across AWS, GCP,
Kubernetes, Azure, Heroku, Vault, and Terraform helper modules.

## What's Changed

### Toolchain And CI

* Raise the module directive to Go 1.26.2.
* Move GitHub Actions workflows onto Node.js 24-compatible actions.
* Split Go validation into tidy, build, test, and vet checks.
* Add non-blocking `govulncheck` module scanning.
* Pin lint tooling so required PR checks do not drift during dependency cleanup.

### Dependency And Provider Maintenance

* Replace Dependabot version-update setup with Renovate provider-scoped groups.
* Refresh security-sensitive Go modules, including Vault API, gRPC, Go JOSE,
  SAML/XML signature, logrus, go-getter, and xz.
* Update AWS SDK v2 service modules and migrate APIGatewayV2 off AWS SDK v1.
* Update GCP/OpenTelemetry and Kubernetes client libraries.
* Refresh smaller provider dependencies, including Heroku and Azure helpers.

### Repository Maintenance

* Add repository agent guidance for future maintenance work.
* Refresh repository maintenance status docs after the standalone fork migration.
* Add release-note categories, PR auto-labeling, and immutable-release guidance for
  future releases.

**Full Changelog**: https://github.com/chenrui333/terraformer/compare/0.8.30...0.9.0
