# Repository Guidance

## Operating Context

- The default branch is `main`.
- This repository is standalone. When using GitHub CLI, pass `-R chenrui333/terraformer` if there is any chance the local remote context could resolve to another repository.
- Use Conventional Commit subjects for commits and pull request titles, for example `chore(deps): ...` or `docs: ...`.
- Keep branch names task-specific. Do not use `codex` in branch names or commit messages.
- Use pull requests for changes to `main`. Branch protection expects the lint and test checks to pass before merging.

## Dependency Automation

- Renovate is the dependency automation path for Go modules and GitHub Actions.
- Do not reintroduce overlapping Dependabot version updates unless explicitly requested.
- Keep Renovate changes small and auditable: adjust `renovate.json`, validate it, and let Renovate regenerate dependency PRs from `main`.
- For large Go dependency backlogs, prefer grouped provider-family updates over broad manual all-at-once bumps.

## Provider Maintenance Guidance

- For provider-support work, read `CLAUDE.md` in addition to this file; it contains the detailed Terraformer provider-maintenance rules.
- New external providers should land the full Terraformer surface together: CLI command, provider and service registration, provider-source mapping, docs, README/provider list when applicable, and tests.
- Use upstream service or provider API clients for discovery, then seed provider-compatible state for refresh. Do not use Terraform provider refresh/import as the inventory discovery mechanism.
- Keep generated HCL secret-free. Prefer environment variables, profiles, or existing provider config paths for authentication, and keep refresh-time auth config separate from generated provider data.
- Apply filters before broad, expensive, or permission-sensitive reads. Skip system, internal, default, or provider-managed resources unless explicitly filtered and verified as safely user-owned.
- Preserve required identity and shape fields through refresh/import fallback. Defer resources with unrecoverable write-only fields, importer mismatches, or unreadable required config into unsupported metadata with evidence.
- Before closing a provider gap issue, compare provider registration, docs, unsupported metadata, and the issue's resource buckets; detect stale or superseded PR branches after parallel provider lanes merge.
- Preserve empty-but-meaningful provider fields, nested state, and source variant markers required for refresh-stable HCL. Reject discovered resources whose source shape is not supported by the Terraform provider.
- Import durable configuration metadata only. Classify high-cardinality row, item, event, or runtime data as unsupported or deferred unless provider read can reconstruct stable configuration.

## Validation

Use the narrowest validation that matches the change, and broaden it when touching shared behavior or dependencies.

- Renovate config: `npx --yes --package renovate renovate-config-validator renovate.json`
- JSON config formatting: `jq . renovate.json`
- Go module changes: `go mod tidy`
- Build: `go build -v`
- Tests: `go test ./...`
- Static diff check: `git diff --check`

## Workflow Notes

- Prefer `bash` for non-trivial multi-line shell scripts and loops.
- Use `rg` for repository searches.
- Keep dependency automation changes separate from feature, bug, or cleanup changes.
- Before opening a PR, confirm the branch is based on `main` and that the PR base is `main`.
