# CI Performance History

This document records durable context from the CI and release performance work
that led to the current shard layout. It is a curated history, not an archive of
scratch reports or raw CI logs.

For the current shard graph, helper modes, invariants, and measurement workflow,
see [CI Build Performance Notes](ci-build-performance.md).

## Decision Timeline

| PR / period | Change | Reason | Behavior impact | Follow-up |
| --- | --- | --- | --- | --- |
| Initial build audit | Kept the repo on native Go and GitHub Actions while measuring bottlenecks | Local and CI evidence showed dependency-sensitive PR validation and release fanout were the visible bottlenecks | No build-system migration | Revisit build-system migration only after native workflow, cache, and release-artifact work no longer explain the cost |
| Release build foundation | Moved provider-specific release builds into isolated temporary workspaces | The old release builder mutated tracked source files while producing provider binaries, which made release builds risky and less cache-friendly | Public artifact names and release layout stayed the same | Continue measuring release fanout from real snapshot and tag runs |
| Release asset instrumentation | Added release staging, count, size, checksum, and publish-preparation timing | Release runs can produce hundreds of assets and many GiB of artifacts, so fan-in needed phase-level timing | Release behavior stayed the same | Use post-instrumentation release data before tuning artifact movement or upload behavior |
| Preflight classification hardening | Made quick-mode diff classification fail closed | A diff or classification failure must not become a clean preflight skip | Dependency-sensitive validation safety improved | Keep classifier errors fail-closed for future path-sensitive narrowing |
| PR core test timing | Added package timing for the regular PR core test path | The test (ubuntu-latest) job was expensive, but package-level cost was opaque | Test coverage stayed the same | Use package timing before sharding core tests |
| Preflight build shard | Split non-fixture package build into a PR-only shard | The build phase dominated provider dependency preflight | Push and full preflight behavior stayed unchanged | Re-measure critical path before more build sharding |
| Provider/cmd test shards | Split provider and command tests out of validation, then into deterministic a, b, and cmd shards | Provider/cmd tests were large enough to hide other validation phases | Test coverage is preserved by shard coverage checks | Keep shard package union equal to the original provider/cmd package list |
| Govulncheck stabilization | Avoided high-fanout root/cmd symbol scans in the source vulnerability workflow | CI showed runner pressure from traversing the full command/provider graph | Source scan remains blocking while avoiding known runner-cancel pressure | Revisit only with new govulncheck timing or failure evidence |
| Vet shard | Split dependency-sensitive go vet into a dedicated PR-only shard | After earlier splits, vet dominated the remaining provider validation job | Vet package coverage stayed the same | Do not duplicate vet package selection in workflow YAML |
| Command test shard | Split command package tests into their own deterministic shard | Command packages were the long pole inside provider test shard a | Provider/cmd shard coverage now includes a, b, and cmd | Keep cmd dedicated unless later timing proves a better split |
| Terraform provider compatibility shard | Split Terraform provider compatibility into a dedicated PR-only shard | Compatibility checks dominated the remaining validation job | Full compatibility coverage stayed intact | Do not split compatibility further from one near-tied run |
| Current docs cleanup | Recorded the current shard graph, invariants, and plateau decision | The workflow reached a near-tie plateau where more generic sharding had weaker risk/reward | Documentation only | Study provider-only path-sensitive narrowing after more completed runs |
| Provider-only narrowing study | Studied strict provider-only preflight narrowing and deferred it | Only 1 of the last 100 merged PRs qualified, below the threshold of at least 3 recent qualifying PRs; preflight-only narrowing was capped by test (ubuntu-latest), so the current upside was about 8 seconds | No behavior change | Revisit only when recent PR history has 3 to 5 isolated provider-only PRs and preflight shards materially exceed test (ubuntu-latest) |

## Timing Baselines

The original release audit found a normal snapshot-style release path around
49 minutes. A representative release produced 299 assets totaling about
17.16 GiB, making artifact staging, checksums, and upload behavior material
release-path concerns.

The latest completed post-sharding dependency-sensitive PR run showed a
near-tie plateau rather than one dominant job:

| Job | Duration seconds |
| --- | ---: |
| Terraform provider compatibility | 991 |
| test (ubuntu-latest) | 983 |
| Preflight build packages | 943 |
| Provider dependency tests cmd | 903 |
| Provider dependency vet | 883 |
| Provider dependency validation | 152 |

In that run, the top three jobs were within 48 seconds and the top five jobs
were within 108 seconds. That is why generic sharding is paused until more
completed dependency-sensitive runs show a clearer long pole.

## Report Classification

Local scratch reports were reviewed and distilled into durable documentation.
The raw reports are not canonical and should not be checked in as-is.

| Report group | Classification | Durable content kept |
| --- | --- | --- |
| Initial build-performance audit | Durable | Native Go/GitHub Actions first, build-system migration deferred, release fanout baseline |
| Provider release build hermeticization | Durable | Release builds should not mutate the checked-out source tree |
| Release performance and asset staging reports | Durable with superseded details | Release asset timing exists, staging must fail closed, release optimization should wait for real post-instrumentation runs |
| Early preflight instrumentation reports | Superseded | Phase timing and fail-closed quick mode are now in the current CI docs |
| PR CI sharding reports through the Terraform compatibility shard | Durable timeline | Sequence of build, test, vet, command, and compatibility shards |
| Incomplete post-sharding run notes | Transient | Superseded by the completed post-sharding plateau measurement |
| Docs cleanup reports | Superseded | Current docs are the durable record |

## Current Decision

Do not add another generic shard from the current data. The top required jobs
are close enough that another shard is likely to add runner scheduling,
branch-protection, and maintenance overhead unless future runs show a clearer
long pole.

Path-sensitive provider-only narrowing was studied and deferred. Only 1 of the
last 100 merged PRs qualified as strict provider-only, below the threshold of
at least 3 recent qualifying PRs. The current preflight-only upside was also
capped by `test (ubuntu-latest)`: even perfect preflight narrowing would move
the observed critical path only from about 991 seconds to about 983 seconds.

Revisit provider-only narrowing only when recent PR history has 3 to 5 isolated
provider-only PRs and completed CI data shows provider preflight shards
materially exceeding `test (ubuntu-latest)`.

Bazel, Buck2, BuildBuddy, Dagger, Earthly, Pants, Nix, or another build system
remain premature. Native Go and GitHub Actions still expose actionable
bottlenecks through timing, sharding, and path-sensitive validation.

## Provider-Only Narrowing Study

Future provider-only narrowing should force full validation for:

- go.mod and go.sum
- .github/**
- .goreleaser*
- build/**
- cmd/root.go
- cmd/provider_cmd_*
- shared package changes
- workflow or script changes
- mixed changes
- unknown paths
- diff or classification failure

Focused validation should be considered only for clearly isolated
providers/<name>/... changes. The classifier must fail closed, explain the
selected mode in the job summary, and include smoke coverage for isolated
provider-only, multiple-provider, mixed, shared, and failure cases.

## Release Follow-Up

Release performance should be revisited after a real post-instrumentation
snapshot or tag release completes.

Known release-path guardrails:

- Provider release builds use isolated temporary workspaces.
- Release asset staging emits timing and size/count summaries.
- Staging failures must fail closed, including traversal failures after partial
  output.
- Public asset names and release layout should remain stable unless a release
  compatibility change is explicitly planned.

## Cleanup Guidance

Use this file and [CI Build Performance Notes](ci-build-performance.md) as the
canonical performance documentation. Do not check in raw scratch reports,
pending CI statuses, machine-local paths, duplicated PR bodies, or stale
recommendations that have already been superseded.
