#!/usr/bin/env bash
set -euo pipefail

MODE="${MODE:-snapshot}"
PROVIDER_DIR="${PROVIDER_DIR:-.goreleaser-extra/provider-binaries}"
ALL_DIR="${ALL_DIR:-.goreleaser-extra/all-binaries}"
ASSET_DIR="${ASSET_DIR:-.goreleaser-extra/release-assets}"
PHASE_ROWS=""
SCRIPT_STARTED_AT="$(date +%s)"

fail() {
  printf 'release-prebuilt-assets: %s\n' "$*" >&2
  exit 1
}

phase_error() {
  printf 'release-prebuilt-assets: %s\n' "$*" >&2
  return 1
}

section() {
  printf '\n==> %s\n' "$*"
}

markdown_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//|/\\|}"
  value="${value//$'\n'/ }"
  printf '%s\n' "$value"
}

asset_count() {
  [[ -d "$ASSET_DIR" ]] || {
    printf '0\n'
    return 0
  }

  find "$ASSET_DIR" -maxdepth 1 -type f | wc -l | tr -d ' '
}

file_size_bytes() {
  local file="$1"

  if stat -c %s "$file" >/dev/null 2>&1; then
    stat -c %s "$file"
  else
    stat -f %z "$file"
  fi
}

asset_bytes() {
  local file size total
  total=0

  [[ -d "$ASSET_DIR" ]] || {
    printf '0\n'
    return 0
  }

  while IFS= read -r -d '' file; do
    size="$(file_size_bytes "$file")" || return 1
    total="$((total + size))"
  done < <(find "$ASSET_DIR" -maxdepth 1 -type f -print0)

  printf '%s\n' "$total"
}

record_phase() {
  local phase="$1"
  local status="$2"
  local duration="$3"
  local count="$4"
  local bytes="$5"
  local notes="$6"

  PHASE_ROWS+="| $(markdown_escape "$phase") | $status | $duration | $count | $bytes | $(markdown_escape "$notes") |"$'\n'
}

write_step_summary() {
  local exit_code="$1"
  local total_duration
  total_duration="$(($(date +%s) - SCRIPT_STARTED_AT))"

  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  {
    printf '## Release Asset Timing\n\n'
    printf -- '- Mode: %s\n' "$MODE"
    printf -- '- Result: %s\n' "$([[ "$exit_code" -eq 0 ]] && printf success || printf failed)"
    printf -- '- Total duration seconds: %s\n' "$total_duration"
    printf -- '- Final artifact count: %s\n' "$(asset_count)"
    printf -- '- Final artifact bytes: %s\n\n' "$(asset_bytes)"
    printf '| Phase | Status | Duration seconds | Artifact count | Total bytes | Notes |\n'
    printf '| --- | --- | ---: | ---: | ---: | --- |\n'
    if [[ -n "$PHASE_ROWS" ]]; then
      printf '%s' "$PHASE_ROWS"
    else
      printf '| script startup | failed | 0 | 0 | 0 | no phases completed |\n'
    fi
  } >>"$GITHUB_STEP_SUMMARY"
}

on_exit() {
  local exit_code="$?"
  write_step_summary "$exit_code" || true
  exit "$exit_code"
}

time_phase() {
  local phase="$1"
  local notes="$2"
  shift 2

  section "$phase"
  local started_at ended_at duration status rc count bytes
  started_at="$(date +%s)"
  set +e
  "$@"
  rc="$?"
  set -e
  ended_at="$(date +%s)"
  duration="$((ended_at - started_at))"
  count="$(asset_count)"
  bytes="$(asset_bytes)"

  if [[ "$rc" -eq 0 ]]; then
    status="success"
  else
    status="failed"
  fi

  printf 'release-prebuilt-assets: timing phase=%q status=%s duration=%ss artifacts=%s bytes=%s notes=%q\n' \
    "$phase" "$status" "$duration" "$count" "$bytes" "$notes"
  record_phase "$phase" "$status" "$duration" "$count" "$bytes" "$notes"

  return "$rc"
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

prepare_asset_dir() {
  rm -rf "$ASSET_DIR" && mkdir -p "$ASSET_DIR"
}

stage_asset_file() {
  local source="$1"
  local destination

  destination="$ASSET_DIR/$(basename "$source")"

  rm -f "$destination"
  if ln "$source" "$destination" 2>/dev/null; then
    return 0
  fi

  cp "$source" "$destination"
}

stage_matching_assets() {
  local source_dir="$1"
  local pattern="$2"
  local file_list
  local file
  local stage_status

  file_list="$(mktemp "${TMPDIR:-/tmp}/terraformer-release-assets.XXXXXX")" || return 1

  if find "$source_dir" -type f -name "$pattern" -print0 >"$file_list"; then
    :
  else
    local find_status="$?"
    rm -f "$file_list"
    return "$find_status"
  fi

  stage_status=0
  while IFS= read -r -d '' file; do
    if ! stage_asset_file "$file"; then
      stage_status=1
      break
    fi
  done <"$file_list"

  rm -f "$file_list"
  return "$stage_status"
}

stage_provider_assets() {
  [[ -d "$PROVIDER_DIR" ]] || {
    phase_error "missing provider binary directory: $PROVIDER_DIR"
    return 1
  }

  stage_matching_assets "$PROVIDER_DIR" 'terraformer-*'
}

stage_all_assets() {
  [[ -d "$ALL_DIR" ]] || {
    phase_error "missing all-in-one binary directory: $ALL_DIR"
    return 1
  }

  stage_matching_assets "$ALL_DIR" 'terraformer-all-*'
}

verify_expected_counts() {
  local provider_count all_count
  provider_count="$(find "$ASSET_DIR" -type f -name 'terraformer-*' ! -name 'terraformer-all-*' | wc -l | tr -d ' ')"
  all_count="$(find "$ASSET_DIR" -type f -name 'terraformer-all-*' | wc -l | tr -d ' ')"

  [[ "$provider_count" -gt 0 ]] || {
    phase_error "no provider binaries staged"
    return 1
  }
  [[ "$all_count" -eq 4 ]] || {
    phase_error "expected 4 all-in-one binaries, found $all_count"
    return 1
  }

  printf 'Staged %s provider binaries and %s all-in-one binaries\n' "$provider_count" "$all_count"
}

list_staged_assets() {
  find "$ASSET_DIR" -maxdepth 1 -type f -print | sort
}

print_checksums() {
  local checksum_name="$1"

  cat "$ASSET_DIR/$checksum_name"
}

print_asset_size() {
  if command -v du >/dev/null 2>&1; then
    du -sh "$ASSET_DIR"
  else
    printf 'du is unavailable; skipped asset size summary\n'
  fi
}

publish_release() {
  local tag="${RELEASE_REF:-${GITHUB_REF_NAME:-}}"
  [[ -n "$tag" ]] || {
    phase_error "RELEASE_REF or GITHUB_REF_NAME is required for release mode"
    return 1
  }
  [[ "$tag" == v* ]] || {
    phase_error "release mode requires a v-prefixed tag, got: $tag"
    return 1
  }

  if gh release view "$tag" >/dev/null 2>&1; then
    local is_draft
    is_draft="$(gh release view "$tag" --json isDraft --jq '.isDraft')" || return 1
    [[ "$is_draft" == "true" ]] || {
      phase_error "release $tag already exists and is not a draft"
      return 1
    }
    gh release delete "$tag" --yes || return 1
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

trap on_exit EXIT

version="$(version_from_ref)"
checksum_name="terraformer_${version}_checksums.txt"

time_phase "Prepare asset directory" "reset $ASSET_DIR" prepare_asset_dir
time_phase "Stage provider binaries" "hardlink/copy terraformer-* from $PROVIDER_DIR" stage_provider_assets
time_phase "Stage all-in-one binaries" "hardlink/copy terraformer-all-* from $ALL_DIR" stage_all_assets
time_phase "Verify expected counts" "require provider binaries and 4 all-in-one binaries" verify_expected_counts
time_phase "Staged assets" "list staged release assets" list_staged_assets
time_phase "Checksum generation" "write $checksum_name" checksum_assets "$checksum_name"
time_phase "Checksums" "print $checksum_name" print_checksums "$checksum_name"
time_phase "Asset size" "du -sh $ASSET_DIR" print_asset_size

if [[ "$MODE" == "release" ]]; then
  time_phase "Publish draft release" "gh release create draft for ${RELEASE_REF:-${GITHUB_REF_NAME:-}}" publish_release
fi
