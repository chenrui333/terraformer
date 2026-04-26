# Release Workflow

Use this checklist when cutting a new Terraformer release.

## Inputs

- Decide the semantic version bump before release prep starts.
- Review the current milestone or tracking issue for release-specific blockers.
- Review merged PRs since the previous release and keep the changelog focused on
  user-visible or maintainer-relevant changes.
- Make sure `main` is current and the worktree is clean before publishing.

For `0.10.0`, treat the release as a minor release because it contains broad
provider dependency modernization after `0.9.0`. Use issue #155 for follow-up
tracking of the larger provider SDK and Terraform-core migrations that should
not block this release once validation is green.

## Checklist

1. Update [CHANGELOG.md](CHANGELOG.md) with the new version entry.
   - Keep the opening paragraph short and release-oriented.
   - Prefer important toolchain, provider, security, and maintenance changes over
     pasting every generated PR line.
   - Keep the compare link at the end of the entry.
2. Preview GitHub-generated notes for comparison:
   ```sh
   VERSION=0.10.0
   PREVIOUS_VERSION=0.9.0
   gh api repos/chenrui333/terraformer/releases/generate-notes \
     -f tag_name="$VERSION" \
     -f target_commitish=main \
     -f previous_tag_name="$PREVIOUS_VERSION"
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
4. Run the GoReleaser config and snapshot preflight before publishing:
   ```sh
   goreleaser check
   goreleaser release --snapshot --clean --skip=publish
   ```
   You can also run the `release` workflow manually to exercise the same
   snapshot path in GitHub Actions.
5. Confirm the GitHub release body, tag, and artifact list are final.
6. Create and push the release tag from the intended `main` commit:
   ```sh
   VERSION=0.10.0
   git fetch origin main --tags
   git checkout main
   git pull --ff-only origin main
   git tag -a "$VERSION" -m "$VERSION"
   git push origin "$VERSION"
   ```
   The tag push runs GoReleaser and creates a draft GitHub release.
7. Review the draft release, then publish it once the notes and assets are
   final.
8. Verify the published release:
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
- Keep draft/preflight checks ahead of publish so the final release has no asset
  or note churn.

## Notes

- Terraformer version tags use plain version tags such as `0.9.0`.
- GoReleaser creates draft releases for manual review before publication.
- The GoReleaser config preserves the existing binary asset names used by README
  install snippets and downstream packaging.
