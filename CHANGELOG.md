# Changelog

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
