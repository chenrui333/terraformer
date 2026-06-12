# CI Build Performance Notes

This repository has a large Go dependency graph. CI performance changes should be measured from GitHub Actions data before validation is removed, gated, or sharded.

## Current PR Critical Path Model

The pull request `tests` workflow keeps dependency-sensitive validation split into visible shards:

- `preflight build packages`: runs `go mod tidy`, lists non-fixture packages, and builds the selected package list.
- `provider dependency tests`: runs provider and command package tests in deterministic shards.
- `provider dependency validation`: runs the remaining preflight checks, including utility tests, vet, static diff, and Terraform compatibility.
- `provider dependency preflight`: aggregates the shard results into one required status.
- `test (ubuntu-latest)`: runs the regular PR core test path and package timing summary.

Push, scheduled, and workflow-dispatch runs still use the full workflow shape rather than the PR-only shard split.

## Measurement Workflow

Use recent Actions runs before choosing an optimization:

```bash
gh run list -R chenrui333/terraformer --workflow tests --limit 80
gh run view -R chenrui333/terraformer <run-id> --json jobs,conclusion,createdAt,updatedAt
gh run view -R chenrui333/terraformer <run-id> --job <job-id> --log
```

Record:

- total workflow duration
- `test (ubuntu-latest)` duration
- preflight shard durations
- phase timing rows from `provider-dependency-preflight.sh`
- package timing rows from `pr-core-test.sh`
- setup-go cache hit or miss behavior

Prefer completed dependency-sensitive PR runs. Do not infer phase timing from cancelled runs.

## Shard Safety Invariants

Sharding is allowed only when the union of shard package lists matches the original package list and no package appears in more than one shard.

Provider and command test shards are checked by:

```bash
MODE=full ONLY_CHECK_PROVIDER_COMMAND_TEST_SHARDS=1 bash .github/scripts/provider-dependency-preflight.sh
```

The check validates that:

- shard `a` and shard `b` are both non-empty
- shard package lists do not overlap
- the sorted union equals `go list ./providers/... ./cmd/...`

Non-fixture build package selection must continue to use packages with non-test Go files and must keep fixture packages out of the build target set.

## Change Guidelines

- Preserve validation on dependency-sensitive PRs.
- Fail closed when path or package classification fails.
- Keep push, scheduled, workflow-dispatch, and release behavior unchanged unless separately measured.
- Prefer two or three deterministic shards over many dynamic shards.
- Keep the aggregate required check clear for branch protection.
- Add timing or shard coverage checks before removing or narrowing validation.

## When To Revisit

Re-run the measurement workflow when any of these change:

- provider count grows materially
- dependency updates regularly exceed the PR critical path budget
- Go version or setup-go cache behavior changes
- package timing summaries show a new dominant package group
- release or provider build jobs start competing with PR validation for runner time

Bazel, BuildBuddy, Buck2, or another build system should remain a later spike unless native workflow sharding and cache behavior no longer explain the bottleneck.
