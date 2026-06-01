# Release Workflow

Use this checklist when cutting a new Terraformer release.

## Inputs

- Decide the semantic version bump before release prep starts.
- Review the current milestone or tracking issue for release-specific blockers.
- Review merged PRs since the previous release and keep the changelog focused on
  user-visible or maintainer-relevant changes.
- Make sure `main` is current and the worktree is clean before publishing.
- Normal provider dependency PRs are covered by the
  `provider-dependency-preflight` CI job. The blocking source vulnerability scan
  runs separately in `govulncheck.yml`.

## Checklist

1. Update [CHANGELOG.md](CHANGELOG.md) with the new version entry.
   - Keep the opening paragraph short and release-oriented.
   - Prefer important toolchain, provider, security, and maintenance changes over
     pasting every generated PR line.
   - Keep the compare link at the end of the entry.
2. Preview GitHub-generated notes for comparison:
   ```sh
   VERSION=v0.11.0
   # Use the actual previous tag name (0.10.0 has no v prefix).
   PREVIOUS_VERSION=0.10.0
   gh api repos/chenrui333/terraformer/releases/generate-notes \
     -f tag_name="$VERSION" \
     -f target_commitish=main \
     -f previous_tag_name="$PREVIOUS_VERSION"
   ```
3. Run the provider dependency and release preflight:
   ```sh
   MODE=release bash .github/scripts/provider-dependency-preflight.sh
   ```
  This verifies `go mod tidy` output, full build coverage, provider and command
  tests, `go vet`, provider/state compatibility scripts, blocking govulncheck
  source package scan, and GoReleaser config.
  `MODE=full` intentionally excludes govulncheck by default because source
  package scanning is handled by the dedicated govulncheck workflow and release
  mode. Set `GOVULNCHECK_SCAN_LEVEL=symbol` when a deeper local symbol scan is
  needed and the runtime cost is acceptable. Set `RUN_GORELEASER_SNAPSHOT=1`
  only when a local monolithic snapshot is practical.
4. Run the `release` workflow manually with an empty `release_tag` to exercise
   the fanout/fanin snapshot path in GitHub Actions.
5. Confirm the GitHub release body, tag, and artifact list are final.
6. Create and push the release tag from the intended `main` commit:
   ```sh
   VERSION=v0.11.0
   git fetch origin main --tags
   git checkout main
   git pull --ff-only origin main
   git tag -a "$VERSION" -m "$VERSION"
   git push origin "$VERSION"
   ```
   The tag push runs the release fanout/fanin workflow and creates a draft
   GitHub release from prebuilt assets.
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

- Terraformer version tags use the `v` prefix (e.g. `v0.11.0`) for Go module
  compatibility. Releases before `v0.11.0` used plain tags (`0.9.0`, `0.10.0`).
- The release workflow creates draft releases for manual review before
  publication.
- The GoReleaser config check and release asset assembly preserve the existing
  binary asset names used by README install snippets and downstream packaging.
