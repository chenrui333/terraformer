# Contributing

Terraformer is maintained as a standalone GitHub repository. Contributions are
welcome when they are focused, reviewable, and aligned with the existing provider
boundaries.

## Pull Requests

- Open pull requests against `main`.
- Use conventional commit style for PR titles, such as `fix(aws): ...`,
  `deps(kubernetes): ...`, `docs: ...`, or `ci: ...`.
- Keep each PR focused on one provider, subsystem, bug, or maintenance task.
- Include a concise summary and any important notes or upstream references in
  the PR body.

## Validation

Run the relevant focused checks for the change, and prefer the existing full
validation set before merge:

- `GOWORK=off go mod tidy`
- `git diff --exit-code -- go.mod go.sum`
- `GOWORK=off go build -v ./...`
- `GOWORK=off go test ./... -count=1`
- `GOWORK=off go vet ./...`
- `git diff --check`

For release workflow changes, also run:

- `GOWORK=off go run github.com/goreleaser/goreleaser/v2@v2.15.4 check`

## Dependency Updates

Provider dependencies should be updated in small, provider-scoped PRs when
possible. The goal is for each PR to make the affected provider or subsystem easy
to identify and test.
