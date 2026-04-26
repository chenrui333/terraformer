# Release Workflow

Use this checklist when cutting a new Terraformer release.

## Inputs

- Decide the semantic version bump before release prep starts.
- Review the current milestone or tracking issue for release-specific blockers.
- Review merged PRs since the previous release and keep the changelog focused on
  user-visible or maintainer-relevant changes.
- Make sure `main` is current and the worktree is clean before publishing.

For `0.9.0`, use tracking issue #120 and treat the release as a minor release
because the Go toolchain floor moved to Go 1.26.2.

## Checklist

1. Update [CHANGELOG.md](CHANGELOG.md) with the new version entry.
   - Keep the opening paragraph short and release-oriented.
   - Prefer important toolchain, provider, security, and maintenance changes over
     pasting every generated PR line.
   - Keep the compare link at the end of the entry.
2. Preview GitHub-generated notes for comparison:
   ```sh
   gh api repos/chenrui333/terraformer/releases/generate-notes \
     -f tag_name=0.9.0 \
     -f previous_tag_name=0.8.30
   ```
3. Run the local validation set:
   ```sh
   GOWORK=off go mod tidy
   git diff --exit-code -- go.mod go.sum
   GOWORK=off go build -v ./...
   GOWORK=off go test ./... -count=1
   GOWORK=off go vet ./...
   git diff --check
   ```
4. If GoReleaser is configured, run the release preflight before publishing:
   ```sh
   goreleaser check
   goreleaser release --snapshot --clean --skip=publish
   ```
5. Confirm the GitHub release body, tag, and artifact list are final.
6. Publish the release through the release workflow.
7. Verify the published release:
   - the tag points at the intended `main` commit
   - the release notes match [CHANGELOG.md](CHANGELOG.md)
   - expected artifacts and checksums are attached
   - install snippets still match the published artifact names

## Immutable Releases

Release immutability is enabled for this repository. Treat release tags, notes,
and assets as final before publishing.

- Do not publish a release expecting to replace assets or retarget the tag later.
- If a published release is wrong, prefer a follow-up release over mutating the
  existing one.
- Keep draft/preflight checks ahead of publish so the final release is boring.

## Notes

- Terraformer version tags use plain version tags such as `0.9.0`.
- The first GoReleaser migration should preserve the existing binary asset names
  used by README install snippets and downstream packaging.
