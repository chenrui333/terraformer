# CI Build Performance Notes

Terraformer has a large Go dependency graph. CI performance changes should be
measured from completed GitHub Actions runs before validation is removed, gated,
or sharded.

This document records the current native Go/GitHub Actions layout and the
guardrails for future performance work. It intentionally does not propose a
Bazel, Buck2, BuildBuddy, Dagger, Earthly, Pants, or Nix migration.

## Current PR Shard Graph

The pull request `tests` workflow keeps dependency-sensitive validation split
into visible PR-only shards. Push, scheduled, and workflow-dispatch runs still
use the full workflow shape rather than this PR-only shard split.

| Job | Owns | Preflight helper mode |
| --- | --- | --- |
| `preflight build packages` | `go mod tidy`, `go.mod` / `go.sum` diff check, non-fixture package listing, and non-fixture package build | `MODE=quick ONLY_BUILD_NON_FIXTURE=1` |
| `provider dependency tests (a)` | Provider/cmd test shard `a` | `MODE=quick ONLY_PROVIDER_COMMAND_TESTS=1 PROVIDER_COMMAND_TEST_SHARD=a` |
| `provider dependency tests (b)` | Provider/cmd test shard `b` | `MODE=quick ONLY_PROVIDER_COMMAND_TESTS=1 PROVIDER_COMMAND_TEST_SHARD=b` |
| `provider dependency tests (cmd)` | Command package test shard | `MODE=quick ONLY_PROVIDER_COMMAND_TESTS=1 PROVIDER_COMMAND_TEST_SHARD=cmd` |
| `provider dependency vet` | Dependency-sensitive `go vet` package set | `MODE=quick ONLY_VET_DEPENDENCY_PACKAGES=1` |
| `provider dependency Terraform provider compatibility` | Provider registry compatibility test through `.github/scripts/terraform-provider-compat.sh` | `MODE=quick ONLY_TERRAFORM_PROVIDER_COMPATIBILITY=1` |
| `provider dependency validation` | Shard coverage check, environment diagnostics, utility tests, static diff, Terraform state compatibility, and skip markers for phases owned by other shards | `MODE=full ONLY_CHECK_PROVIDER_COMMAND_TEST_SHARDS=1`, then `MODE=quick` with skip flags |
| `provider dependency preflight` | Aggregates the PR-only shard results into one required status | Workflow-only aggregate |
| `test (ubuntu-latest)` | Regular PR core package tests plus package timing summary | `.github/scripts/pr-core-test.sh` |

The aggregate job must require every PR-only shard that owns part of provider
dependency validation. It should stay small and only report shard results.

## Helper Modes

`.github/scripts/provider-dependency-preflight.sh` supports these modes and
environment variables:

- `MODE=quick`: classify the PR diff before running dependency-sensitive
  checks. Dependency-sensitive paths run validation, no-match diffs skip
  validation, and diff/classification errors fail closed.
- `MODE=full`: run the full provider dependency preflight. This is used for
  push, scheduled, and workflow-dispatch runs.
- `MODE=release`: run full preflight plus release-only checks.
- `ONLY_BUILD_NON_FIXTURE=1`: run tidy, package listing, and non-fixture
  package build only.
- `ONLY_PROVIDER_COMMAND_TESTS=1`: run provider/cmd package tests only.
- `PROVIDER_COMMAND_TEST_SHARD=all|a|b|cmd`: select the provider/cmd package
  shard. The default is `all`.
- `ONLY_CHECK_PROVIDER_COMMAND_TEST_SHARDS=1`: prove shard `a`, shard `b`,
  and shard `cmd` cover the original provider/cmd package list with no
  duplicates.
- `ONLY_VET_DEPENDENCY_PACKAGES=1`: run dependency-sensitive `go vet` only.
- `ONLY_TERRAFORM_PROVIDER_COMPATIBILITY=1`: run Terraform provider
  compatibility only.
- `ONLY_GOVULNCHECK=1`: run the govulncheck source scan used by
  `.github/workflows/govulncheck.yml`.
- `SKIP_BUILD_NON_FIXTURE=1`, `SKIP_PROVIDER_COMMAND_TESTS=1`,
  `SKIP_VET_DEPENDENCY_PACKAGES=1`, and
  `SKIP_TERRAFORM_PROVIDER_COMPATIBILITY=1`: record that the current PR shard
  is skipping phases already owned by dedicated required shards.

Keep new helper modes narrow. Prefer reusing script functions so PR-only shards
and full preflight stay behaviorally aligned.

## Shard Safety Invariants

Sharding is allowed only when the original package or check coverage is
preserved.

Required invariants:

- The union of shard package lists equals the original package list.
- No package is dropped.
- No package is duplicated unless the duplicate is intentional and documented.
- Empty shards fail unless an empty shard is explicitly expected.
- Fixture/test package exclusions are preserved.
- Diff, path, and package classification errors fail closed.
- Push, scheduled, workflow-dispatch, and release behavior remain unchanged
  unless separately measured and intentionally changed.

Provider and command test shards are checked by:

```bash
MODE=full ONLY_CHECK_PROVIDER_COMMAND_TEST_SHARDS=1 bash .github/scripts/provider-dependency-preflight.sh
```

That check validates that:

- shard `a`, shard `b`, and shard `cmd` are non-empty
- shard package lists do not overlap
- the sorted union equals `go list ./providers/... ./cmd/...`

Non-fixture build package selection must continue to use packages with non-test
Go files and must keep fixture packages out of the build target set. Do not pass
empty packages such as the top-level `providers` package to `go build`.

The PR-only vet shard must keep using `vet_dependency_sensitive_packages` from
`.github/scripts/provider-dependency-preflight.sh`; do not duplicate or narrow
its package list in workflow YAML.

The Terraform provider compatibility shard must keep using
`.github/scripts/terraform-provider-compat.sh` through
`.github/scripts/provider-dependency-preflight.sh`. It is a single
compatibility test over the command/provider graph, so do not split it further
without multiple completed runs proving a clear long pole.

The blocking govulncheck source scan follows the same non-fixture package
boundary. The root CLI entrypoint and `cmd` package are scanned at package
level because symbol-level analysis of those packages traverses the full
command/provider graph and has been runner-canceled in CI.

## Timing Baseline

The latest completed post-#627 dependency-sensitive PR run showed a near-tie
plateau, not a single dominant long pole. Treat this as one data point, not a
stable average.

Run `27440198139`:

| Job | Duration seconds | Dominant phase |
| --- | ---: | --- |
| `provider dependency Terraform provider compatibility` | 991 | Terraform provider compatibility: 913s |
| `test (ubuntu-latest)` | 983 | PR core test packages: 879s |
| `preflight build packages` | 943 | Build non-fixture packages: 815s |
| `provider dependency tests (cmd)` | 903 | cmd package test shard: 852s |
| `provider dependency vet` | 883 | `go vet`: 852s |
| `provider dependency tests (a)` | 688 | shard `a` tests: 645s |
| `provider dependency tests (b)` | 349 | shard `b` tests: 303s |
| `provider dependency validation` | 152 | utility tests, static diff, Terraform state compatibility |
| `provider dependency preflight` | 2 | aggregate only |

Post-#627, the top three required jobs were within 48 seconds and the top five
were within 108 seconds. More generic sharding is therefore likely to add
maintenance and runner-scheduling overhead unless another completed run shows a
clearer long pole.

## Decision Log

- The build/release audit favored native Go and GitHub Actions optimization
  first. Bazel, Buck2, and BuildBuddy remain deferred because native timing,
  sharding, release fanout cleanup, and artifact measurement still explain the
  bottlenecks.
- Provider-specific release builds were changed to use isolated temporary
  workspaces so release builds no longer mutate the checked-out source tree.
- Release asset staging and checksum/publish preparation now emit structured
  timing so release fan-in can be measured from real snapshot or tag runs.
- Provider preflight quick mode now fails closed on diff/classification errors.
- PR core tests, provider preflight phases, provider/cmd test shards, vet, and
  Terraform provider compatibility now report timing at the phase/job boundary.
- After run `27440198139`, pause generic sharding. The next likely useful
  study is path-sensitive provider-only narrowing, but only after more completed
  post-#627 runs confirm the plateau and provider-only PR history shows enough
  upside.

## Measurement Workflow

Use completed Actions runs before choosing another optimization:

```bash
gh run list -R chenrui333/terraformer --workflow tests --limit 80
gh run view -R chenrui333/terraformer <run-id> --json jobs,conclusion,createdAt,updatedAt
gh run view -R chenrui333/terraformer <run-id> --job <job-id> --log
```

Record:

- total workflow duration
- `test (ubuntu-latest)` duration and package timing rows from
  `.github/scripts/pr-core-test.sh`
- preflight shard durations
- phase timing rows from `.github/scripts/provider-dependency-preflight.sh`
- setup-go cache behavior
- runner queue/start skew, because it can hide a near-tie plateau

Prefer completed dependency-sensitive PR runs. Do not infer phase timing from
cancelled runs or still-running jobs.

## Future Provider-Only Narrowing Study

Provider-only narrowing is the next likely optimization direction if future
completed runs keep showing a plateau. Do not implement it until the classifier
and expected savings are proven from real PR history.

Study plan:

1. Classify recent PRs by changed path type.
2. Count isolated provider-only PRs and estimate how often full dependency
   validation could have been narrowed.
3. Design a fail-closed classifier.
4. Add smoke tests for isolated provider-only changes, multiple provider-only
   changes, mixed changes, full-validation paths, and diff failures.
5. Keep one aggregate required status so branch protection remains clear.

Force full validation for:

- `go.mod` or `go.sum`
- `.github/**`
- `.goreleaser.yaml` or other release configuration
- `build/**`
- `cmd/root.go`
- `cmd/provider_cmd_*`
- shared packages such as `terraformutils/**`
- workflow or script changes
- mixed provider/shared changes
- unknown paths
- diff or classification failures

Focused provider-only validation should still run lightweight shared checks and
must explain in the job summary why focused or full mode was selected.

## Release Performance Notes

The release path has separate bottlenecks from PR validation:

- Provider-specific release binaries are built from isolated temporary
  workspaces rather than by mutating tracked source files.
- Provider build logs include structured timing around provider enumeration,
  temporary workspace setup, per-provider builds, and cleanup.
- `.github/scripts/release-prebuilt-assets.sh` emits release asset timing and
  a GitHub Step Summary for staging, count verification, size accounting,
  checksum generation, and publish preparation.
- Staged release assets should fail closed on artifact traversal or staging
  errors. Do not weaken that behavior while optimizing copies or hardlinks.
- Revisit release optimization only after a real post-instrumentation snapshot
  or tag release. Artifact movement, checksum generation, and GitHub release
  upload can dominate release wall time independently of Go compile time.

## Change Guidelines

- Preserve validation on dependency-sensitive PRs.
- Fail closed when path or package classification fails.
- Keep push, scheduled, workflow-dispatch, and release behavior unchanged unless
  separately measured.
- Prefer two or three deterministic shards over many dynamic shards.
- Keep the aggregate required check clear for branch protection.
- Add timing or shard coverage checks before removing or narrowing validation.
- Do not add another generic shard from a single near-tied run.

## When To Revisit

Rerun the measurement workflow when any of these change:

- one or two more completed post-#627 dependency-sensitive PR runs are available
- provider count grows materially
- dependency updates regularly exceed the PR critical path budget
- Go version or setup-go cache behavior changes
- package timing summaries show a new dominant package group
- release or provider build jobs start competing with PR validation for runner
  time
- a real post-instrumentation snapshot or tag release completes

Bazel, BuildBuddy, Buck2, or another build system should remain a later spike
unless native workflow sharding, path-sensitive validation, cache behavior, and
release artifact movement no longer explain the bottleneck.
