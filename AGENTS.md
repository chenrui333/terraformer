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
