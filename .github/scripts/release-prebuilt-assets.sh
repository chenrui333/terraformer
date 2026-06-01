#!/usr/bin/env bash
set -euo pipefail

MODE="${MODE:-snapshot}"
PROVIDER_DIR="${PROVIDER_DIR:-.goreleaser-extra/provider-binaries}"
ALL_DIR="${ALL_DIR:-.goreleaser-extra/all-binaries}"
ASSET_DIR="${ASSET_DIR:-.goreleaser-extra/release-assets}"

fail() {
  printf 'release-prebuilt-assets: %s\n' "$*" >&2
  exit 1
}

section() {
  printf '\n==> %s\n' "$*"
}

version_from_ref() {
  local ref="${RELEASE_REF:-${GITHUB_REF_NAME:-snapshot}}"
  case "$ref" in
    v*) printf '%s\n' "${ref#v}" ;;
    [0-9]*) printf '%s\n' "$ref" ;;
    *) printf 'snapshot\n' ;;
  esac
}

checksum_assets() {
  local checksum_name="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$ASSET_DIR" && sha256sum terraformer-* >"$checksum_name")
  else
    (cd "$ASSET_DIR" && shasum -a 256 terraformer-* >"$checksum_name")
  fi
}

stage_assets() {
  [[ -d "$PROVIDER_DIR" ]] || fail "missing provider binary directory: $PROVIDER_DIR"
  [[ -d "$ALL_DIR" ]] || fail "missing all-in-one binary directory: $ALL_DIR"

  rm -rf "$ASSET_DIR"
  mkdir -p "$ASSET_DIR"

  find "$PROVIDER_DIR" -type f -name 'terraformer-*' -exec cp '{}' "$ASSET_DIR/" \;
  find "$ALL_DIR" -type f -name 'terraformer-all-*' -exec cp '{}' "$ASSET_DIR/" \;

  local provider_count all_count
  provider_count="$(find "$ASSET_DIR" -type f -name 'terraformer-*' ! -name 'terraformer-all-*' | wc -l | tr -d ' ')"
  all_count="$(find "$ASSET_DIR" -type f -name 'terraformer-all-*' | wc -l | tr -d ' ')"

  [[ "$provider_count" -gt 0 ]] || fail "no provider binaries staged"
  [[ "$all_count" -eq 4 ]] || fail "expected 4 all-in-one binaries, found $all_count"

  section "Staged assets"
  find "$ASSET_DIR" -maxdepth 1 -type f -print | sort
}

publish_release() {
  local tag="${RELEASE_REF:-${GITHUB_REF_NAME:-}}"
  [[ -n "$tag" ]] || fail "RELEASE_REF or GITHUB_REF_NAME is required for release mode"
  [[ "$tag" == v* ]] || fail "release mode requires a v-prefixed tag, got: $tag"

  if gh release view "$tag" >/dev/null 2>&1; then
    local is_draft
    is_draft="$(gh release view "$tag" --json isDraft --jq '.isDraft')"
    [[ "$is_draft" == "true" ]] || fail "release $tag already exists and is not a draft"
    gh release delete "$tag" --yes
  fi

  gh release create "$tag" "$ASSET_DIR"/terraformer-* "$ASSET_DIR"/terraformer_*_checksums.txt \
    --draft \
    --title "$tag" \
    --generate-notes
}

case "$MODE" in
  snapshot|release) ;;
  *) fail "unsupported MODE=$MODE; expected snapshot or release" ;;
esac

version="$(version_from_ref)"
checksum_name="terraformer_${version}_checksums.txt"

stage_assets

section "Checksums"
checksum_assets "$checksum_name"
cat "$ASSET_DIR/$checksum_name"

if command -v du >/dev/null 2>&1; then
  section "Asset size"
  du -sh "$ASSET_DIR"
fi

if [[ "$MODE" == "release" ]]; then
  section "Publish draft release"
  publish_release
fi
