#!/usr/bin/env bash
set -euo pipefail

MODE="${MODE:-full}"
export GOWORK=off
PHASE_ROWS=""
SCRIPT_STARTED_AT="$(date +%s)"
DEPENDENCY_SENSITIVE=1
BUILD_PACKAGES=()
PROVIDER_COMMAND_TEST_PACKAGES=()

section() {
  printf '\n==> %s\n' "$1"
}

warn() {
  printf 'warning: %s\n' "$1" >&2
}

fail() {
  printf 'error: %s\n' "$1" >&2
  exit 1
}

case "$MODE" in
  quick|full|release) ;;
  *) fail "MODE must be one of quick, full, or release" ;;
esac

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" ]]; then
  fail "provider dependency preflight must run inside a git repository"
fi
if [[ "$PWD" != "$repo_root" ]]; then
  fail "provider dependency preflight must run from repo root: $repo_root"
fi
if [[ ! -f go.mod || ! -d providers || ! -d cmd ]]; then
  fail "provider dependency preflight must run from the Terraformer repository root"
fi

release_backup_dir=""
restore_release_outputs() {
  if [[ -z "$release_backup_dir" ]]; then
    return
  fi

  rm -rf dist .goreleaser-extra
  if [[ -e "$release_backup_dir/dist" ]]; then
    mv "$release_backup_dir/dist" dist
  fi
  if [[ -e "$release_backup_dir/.goreleaser-extra" ]]; then
    mv "$release_backup_dir/.goreleaser-extra" .goreleaser-extra
  fi
  if [[ -e "$release_backup_dir/cmd" ]]; then
    rm -rf cmd
    cp -a "$release_backup_dir/cmd" cmd
  fi
  rm -rf "$release_backup_dir"
}

markdown_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//|/\\|}"
  value="${value//$'\n'/ }"
  printf '%s\n' "$value"
}

record_phase() {
  local phase="$1"
  local status="$2"
  local duration="$3"
  local notes="$4"

  PHASE_ROWS+="| $(markdown_escape "$phase") | $status | $duration | $(markdown_escape "$notes") |"$'\n'
}

write_step_summary() {
  local exit_code="$1"
  local total_duration
  total_duration="$(($(date +%s) - SCRIPT_STARTED_AT))"

  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  {
    printf '## Provider Dependency Preflight Timing\n\n'
    printf -- '- Mode: %s\n' "$MODE"
    printf -- '- Result: %s\n' "$([[ "$exit_code" -eq 0 ]] && printf success || printf failed)"
    printf -- '- Total duration seconds: %s\n\n' "$total_duration"
    printf '| Phase | Status | Duration seconds | Notes |\n'
    printf '| --- | --- | ---: | --- |\n'
    if [[ -n "$PHASE_ROWS" ]]; then
      printf '%s' "$PHASE_ROWS"
    else
      printf '| script startup | failed | 0 | no phases completed |\n'
    fi
  } >>"$GITHUB_STEP_SUMMARY"
}

on_exit() {
  local exit_code="$?"
  restore_release_outputs || true
  write_step_summary "$exit_code" || true
  exit "$exit_code"
}

time_phase() {
  local phase="$1"
  local notes="$2"
  shift 2

  section "$phase"
  local started_at ended_at duration status rc
  started_at="$(date +%s)"
  set +e
  "$@"
  rc="$?"
  set -e
  ended_at="$(date +%s)"
  duration="$((ended_at - started_at))"

  if [[ "$rc" -eq 0 ]]; then
    status="success"
  else
    status="failed"
  fi

  printf 'provider-dependency-preflight: timing phase=%q status=%s duration=%ss notes=%q\n' \
    "$phase" "$status" "$duration" "$notes"
  record_phase "$phase" "$status" "$duration" "$notes"

  return "$rc"
}

trap on_exit EXIT

prepare_release_output_cleanup() {
  for path in dist .goreleaser-extra; do
    if git ls-files --error-unmatch "$path" >/dev/null 2>&1; then
      fail "refusing to run release preflight with tracked $path output"
    fi
  done

  release_backup_dir="$(mktemp -d "${TMPDIR:-/tmp}/terraformer-release-preflight.XXXXXX")"
  if [[ -e dist ]]; then
    mv dist "$release_backup_dir/dist"
  fi
  if [[ -e .goreleaser-extra ]]; then
    mv .goreleaser-extra "$release_backup_dir/.goreleaser-extra"
  fi
  cp -a cmd "$release_backup_dir/cmd"
}

resolve_base_ref() {
  local ref="${BASE_REF:-origin/main}"
  if git rev-parse --verify --quiet "$ref^{commit}" >/dev/null; then
    printf '%s\n' "$ref"
    return
  fi

  warn "BASE_REF '$ref' was not found; falling back to HEAD~1"
  if git rev-parse --verify --quiet "HEAD~1^{commit}" >/dev/null; then
    printf '%s\n' "HEAD~1"
    return
  fi

  fail "could not resolve BASE_REF '$ref' or fallback HEAD~1"
}

is_dependency_sensitive_path() {
  local path="$1"
  case "$path" in
    go.mod|go.sum|renovate.json|.goreleaser.yaml) return 0 ;;
    .github/workflows/test.yml|.github/workflows/release.yaml|.github/workflows/govulncheck.yml) return 0 ;;
    .github/scripts/*|build/*|providers/*|cmd/*) return 0 ;;
    *) return 1 ;;
  esac
}

has_dependency_sensitive_changes() {
  local base_ref="$1"
  local changed_paths
  local path
  local diff_args=("$base_ref...HEAD")

  if changed_paths="$(git diff --name-only "${diff_args[@]}" 2>/dev/null)"; then
    :
  else
    warn "could not diff $base_ref...HEAD; falling back to $base_ref HEAD"
    if changed_paths="$(git diff --name-only "$base_ref" HEAD)"; then
      :
    else
      return 2
    fi
  fi

  while IFS= read -r path; do
    if [[ -z "$path" ]]; then
      continue
    fi
    if is_dependency_sensitive_path "$path"; then
      return 0
    fi
  done <<<"$changed_paths"

  return 1
}

ensure_govulncheck() {
  local gobin
  gobin="$(go env GOBIN)" || return
  if [[ -z "$gobin" ]]; then
    gobin="$(go env GOPATH)/bin" || return
  fi
  export PATH="$gobin:$PATH"

  if command -v govulncheck >/dev/null 2>&1; then
    return
  fi

  section "Install govulncheck"
  go install golang.org/x/vuln/cmd/govulncheck@v1.3.0
}

run_compat_script() {
  local script="$1"
  local name="$2"

  if [[ -f "$script" && -r "$script" ]]; then
    bash "$script"
    return
  fi

  printf 'Skipping %s; %s is not present/readable.\n' "$name" "$script"
}

go_list_count() {
  local label="$1"
  shift
  local output
  local count

  if output="$("$@" 2>/dev/null)"; then
    if [[ -n "$output" ]]; then
      count="$(printf '%s\n' "$output" | wc -l | tr -d ' ')"
    else
      count=0
    fi
    printf '%s: %s\n' "$label" "$count"
  else
    printf '%s: unavailable (%s failed)\n' "$label" "$*"
  fi
}

environment_diagnostics() {
  local gocache gomodcache gotmpdir tmpfs

  printf 'go version: '
  go version || true

  printf 'go env GOCACHE GOMODCACHE GOTMPDIR GOFLAGS:\n'
  go env -json GOCACHE GOMODCACHE GOTMPDIR GOFLAGS || true

  go_list_count "local packages" go list ./...
  go_list_count "transitive packages" go list -deps ./...

  gocache="$(go env GOCACHE 2>/dev/null || true)"
  gomodcache="$(go env GOMODCACHE 2>/dev/null || true)"
  gotmpdir="$(go env GOTMPDIR 2>/dev/null || true)"

  for cache_dir in "$gocache" "$gomodcache"; do
    if [[ -n "$cache_dir" && -e "$cache_dir" ]]; then
      du -sh "$cache_dir" 2>/dev/null || true
    fi
  done

  printf 'workspace filesystem:\n'
  df -h "$PWD" || true
  tmpfs="${gotmpdir:-${TMPDIR:-/tmp}}"
  if [[ -e "$tmpfs" ]]; then
    printf 'temporary filesystem:\n'
    df -h "$tmpfs" || true
  fi
}

detect_dependency_sensitive_changes() {
  local base_ref
  local sensitive_status

  base_ref="$(resolve_base_ref)" || return
  printf 'Using base ref: %s\n' "$base_ref"
  if has_dependency_sensitive_changes "$base_ref"; then
    DEPENDENCY_SENSITIVE=1
    printf 'Dependency-sensitive changes detected; running provider dependency preflight.\n'
    return
  else
    sensitive_status="$?"
  fi

  if [[ "$sensitive_status" -eq 1 ]]; then
    DEPENDENCY_SENSITIVE=0
    printf 'No dependency-sensitive changes detected; skipping provider dependency preflight.\n'
    return
  fi

  return "$sensitive_status"
}

run_go_mod_tidy_check() {
  go mod tidy || return
  git diff --exit-code -- go.mod go.sum
}

list_build_packages() {
  local package_file
  local package

  package_file="$(mktemp "${TMPDIR:-/tmp}/terraformer-build-packages.XXXXXX")" || return
  if go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... >"$package_file"; then
    :
  else
    local list_status="$?"
    rm -f "$package_file"
    return "$list_status"
  fi

  BUILD_PACKAGES=()
  while IFS= read -r package; do
    [[ -n "$package" ]] || continue
    case "$package" in
      github.com/chenrui333/terraformer/tests/*) continue ;;
    esac
    BUILD_PACKAGES+=("$package")
  done <"$package_file"
  rm -f "$package_file"

  printf 'Selected %s package(s) for build.\n' "${#BUILD_PACKAGES[@]}"
}

build_non_fixture_packages() {
  go build -v "${BUILD_PACKAGES[@]}"
}

validate_provider_command_test_shard() {
  local shard="${1:-all}"

  case "$shard" in
    all|a|b) ;;
    *) fail "PROVIDER_COMMAND_TEST_SHARD must be one of all, a, or b" ;;
  esac
}

provider_command_test_package_group() {
  local package="$1"
  local prefix="github.com/chenrui333/terraformer/providers/"
  local provider

  case "$package" in
    github.com/chenrui333/terraformer/cmd|github.com/chenrui333/terraformer/cmd/*)
      printf 'a\n'
      return
      ;;
    github.com/chenrui333/terraformer/providers)
      printf 'a\n'
      return
      ;;
    "$prefix"*)
      provider="${package#"$prefix"}"
      provider="${provider%%/*}"
      ;;
    *)
      fail "unexpected provider/cmd test package: $package"
      ;;
  esac

  # Keep shard membership explicit and deterministic. The current split balances
  # package count for the measured provider/cmd test path; new providers default to shard b.
  case "$provider" in
    aws|azure|azuredevops|commercetools|datadog|equinixmetal|fastly|github|gmailfilter|helm|honeycombio|ionoscloud|keycloak|launchdarkly|logzio|mikrotik|newrelic|octopusdeploy|opal|opsgenie|panos|tencentcloud|vultr|yandex)
      printf 'a\n'
      ;;
    *)
      printf 'b\n'
      ;;
  esac
}

provider_command_test_package_matches_shard() {
  local package="$1"
  local shard="${2:-all}"
  local package_group

  validate_provider_command_test_shard "$shard"
  if [[ "$shard" == "all" ]]; then
    return 0
  fi

  package_group="$(provider_command_test_package_group "$package")"
  [[ "$package_group" == "$shard" ]]
}

list_provider_command_test_packages() {
  local package_file
  local package
  local shard="${PROVIDER_COMMAND_TEST_SHARD:-all}"

  validate_provider_command_test_shard "$shard"
  package_file="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-packages.XXXXXX")" || return
  if go list ./providers/... ./cmd/... >"$package_file"; then
    :
  else
    local list_status="$?"
    rm -f "$package_file"
    return "$list_status"
  fi

  PROVIDER_COMMAND_TEST_PACKAGES=()
  while IFS= read -r package; do
    [[ -n "$package" ]] || continue
    if provider_command_test_package_matches_shard "$package" "$shard"; then
      PROVIDER_COMMAND_TEST_PACKAGES+=("$package")
    fi
  done <"$package_file"
  rm -f "$package_file"

  if [[ "${#PROVIDER_COMMAND_TEST_PACKAGES[@]}" -eq 0 ]]; then
    fail "provider/cmd test shard $shard selected no packages"
  fi
  printf 'Selected %s provider/cmd test package(s) for shard %s.\n' "${#PROVIDER_COMMAND_TEST_PACKAGES[@]}" "$shard"
}

write_provider_command_test_packages() {
  local shard="$1"
  local output_file="$2"
  local old_shard="${PROVIDER_COMMAND_TEST_SHARD:-}"

  PROVIDER_COMMAND_TEST_SHARD="$shard"
  list_provider_command_test_packages
  printf '%s\n' "${PROVIDER_COMMAND_TEST_PACKAGES[@]}" | sort >"$output_file"

  if [[ -n "$old_shard" ]]; then
    PROVIDER_COMMAND_TEST_SHARD="$old_shard"
  else
    unset PROVIDER_COMMAND_TEST_SHARD
  fi
}

check_provider_command_test_shards() {
  local all_packages shard_a shard_b union duplicates

  all_packages="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-all.XXXXXX")" || return
  shard_a="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-a.XXXXXX")" || return
  shard_b="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-b.XXXXXX")" || return
  union="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-union.XXXXXX")" || return
  duplicates="$(mktemp "${TMPDIR:-/tmp}/terraformer-provider-command-test-duplicates.XXXXXX")" || return

  write_provider_command_test_packages all "$all_packages"
  write_provider_command_test_packages a "$shard_a"
  write_provider_command_test_packages b "$shard_b"

  cat "$shard_a" "$shard_b" | sort >"$union"
  cat "$shard_a" "$shard_b" | sort | uniq -d >"$duplicates"

  if [[ -s "$duplicates" ]]; then
    printf 'Duplicate provider/cmd test packages across shards:\n' >&2
    cat "$duplicates" >&2
    rm -f "$all_packages" "$shard_a" "$shard_b" "$union" "$duplicates"
    return 1
  fi

  if ! diff -u "$all_packages" "$union"; then
    rm -f "$all_packages" "$shard_a" "$shard_b" "$union" "$duplicates"
    return 1
  fi

  printf 'Provider/cmd test shard coverage ok: all=%s shard_a=%s shard_b=%s.\n' \
    "$(wc -l <"$all_packages" | tr -d ' ')" \
    "$(wc -l <"$shard_a" | tr -d ' ')" \
    "$(wc -l <"$shard_b" | tr -d ' ')"
  rm -f "$all_packages" "$shard_a" "$shard_b" "$union" "$duplicates"
}

skip_pr_build_job_validation() {
  printf 'Skipping in this job; the PR preflight build job validates this phase.\n'
}

skip_pr_provider_test_job_validation() {
  printf 'Skipping in this job; the PR provider test job validates this phase.\n'
}

run_build_package_validation() {
  time_phase "Go module tidy" "go mod tidy and go.mod/go.sum diff check" run_go_mod_tidy_check
  time_phase "Package listing" "go list non-fixture packages for build" list_build_packages
  time_phase "Build non-fixture packages" "go build selected non-fixture packages" build_non_fixture_packages
}

test_provider_and_command_packages() {
  list_provider_command_test_packages || return
  go test "${PROVIDER_COMMAND_TEST_PACKAGES[@]}" -count=1
}

run_provider_command_test_validation() {
  local shard="${PROVIDER_COMMAND_TEST_SHARD:-all}"

  time_phase "Test provider and command packages" "go test provider/cmd packages shard=$shard" test_provider_and_command_packages
}

test_build_and_utility_packages() {
  go test ./build/... ./terraformutils/... ./version -count=1
}

vet_dependency_sensitive_packages() {
  go vet ./providers/... ./cmd/... ./build/... ./terraformutils/... ./version
}

static_diff_check() {
  git diff --check
}

run_provider_validation() {
  time_phase "Environment diagnostics" "go version, go env, package counts, cache usage, filesystem space" environment_diagnostics
  if [[ "${SKIP_BUILD_NON_FIXTURE:-0}" == "1" ]]; then
    time_phase "Go module tidy" "validated by the PR preflight build job" skip_pr_build_job_validation
    time_phase "Package listing" "validated by the PR preflight build job" skip_pr_build_job_validation
    time_phase "Build non-fixture packages" "validated by the PR preflight build job" skip_pr_build_job_validation
  else
    run_build_package_validation
  fi
  if [[ "${SKIP_PROVIDER_COMMAND_TESTS:-0}" == "1" ]]; then
    time_phase "Test provider and command packages" "validated by the PR provider test job" skip_pr_provider_test_job_validation
  else
    run_provider_command_test_validation
  fi
  time_phase "Test build and utility packages" "go test ./build/... ./terraformutils/... ./version -count=1" test_build_and_utility_packages
  time_phase "Vet dependency-sensitive packages" "go vet providers, cmd, build, terraformutils, version" vet_dependency_sensitive_packages
  time_phase "Static diff check" "git diff --check" static_diff_check
  time_phase "Terraform state compatibility" "bash .github/scripts/terraform-state-compat.sh if present" run_compat_script ".github/scripts/terraform-state-compat.sh" "Terraform state compatibility"
  time_phase "Terraform provider compatibility" "bash .github/scripts/terraform-provider-compat.sh if present" run_compat_script ".github/scripts/terraform-provider-compat.sh" "Terraform provider compatibility"
}

run_govulncheck_source_scan() {
  local batch=()
  local batch_size="${GOVULNCHECK_BATCH_SIZE:-25}"
  local package
  local package_file=""
  local packages=()
  local scan_level="${GOVULNCHECK_SCAN_LEVEL:-symbol}"

  ensure_govulncheck || return

  case "$scan_level" in
    package|symbol) ;;
    *) fail "GOVULNCHECK_SCAN_LEVEL must be one of package or symbol" ;;
  esac
  if ! [[ "$batch_size" =~ ^[1-9][0-9]*$ ]]; then
    fail "GOVULNCHECK_BATCH_SIZE must be a positive integer"
  fi

  section "govulncheck source scan ($scan_level level)"
  if [[ -n "${GOVULNCHECK_PACKAGES:-}" ]]; then
    read -r -a packages <<<"${GOVULNCHECK_PACKAGES}"
  else
    package_file="$(mktemp "${TMPDIR:-/tmp}/terraformer-govulncheck-packages.XXXXXX")" || return
    if go list ./... >"$package_file"; then
      :
    else
      local list_status="$?"
      rm -f "$package_file"
      return "$list_status"
    fi
    while IFS= read -r package; do
      packages+=("$package")
    done <"$package_file"
    rm -f "$package_file"
  fi

  if [[ "${#packages[@]}" -eq 0 ]]; then
    fail "no Go packages found for govulncheck source scan"
  fi

  for package in "${packages[@]}"; do
    batch+=("$package")
    if [[ "${#batch[@]}" -ge "$batch_size" ]]; then
      printf 'Scanning %s package(s) at %s level: %s\n' "${#batch[@]}" "$scan_level" "${batch[*]}"
      govulncheck -scan="$scan_level" "${batch[@]}" || return
      batch=()
    fi
  done

  if [[ "${#batch[@]}" -gt 0 ]]; then
    printf 'Scanning %s package(s) at %s level: %s\n' "${#batch[@]}" "$scan_level" "${batch[*]}"
    govulncheck -scan="$scan_level" "${batch[@]}"
  fi
}

run_release_validation() {
  if ! command -v goreleaser >/dev/null 2>&1; then
    fail "goreleaser is required for MODE=release; install GoReleaser or run through the release workflow"
  fi

  time_phase "GoReleaser check" "goreleaser check" goreleaser check

  if [[ "${RUN_GORELEASER_SNAPSHOT:-0}" == "1" ]]; then
    prepare_release_output_cleanup
    time_phase "GoReleaser snapshot preflight" "goreleaser release --snapshot --clean --skip=publish" goreleaser release --snapshot --clean --skip=publish --timeout=3h --parallelism=1
  else
    time_phase "GoReleaser snapshot preflight" "skip unless RUN_GORELEASER_SNAPSHOT=1" skip_goreleaser_snapshot_preflight
  fi
}

skip_goreleaser_snapshot_preflight() {
  printf 'Skipping local GoReleaser snapshot by default; the release workflow runs fanout/fanin snapshot validation with prebuilt assets.\n'
  printf 'Set RUN_GORELEASER_SNAPSHOT=1 to run the monolithic local snapshot anyway.\n'
}

if [[ "$MODE" == "quick" ]]; then
  time_phase "Detect dependency-sensitive changes" "diff BASE_REF against HEAD and classify paths" detect_dependency_sensitive_changes
  if [[ "$DEPENDENCY_SENSITIVE" -eq 0 ]]; then
    exit 0
  fi
fi

if [[ "${ONLY_GOVULNCHECK:-0}" == "1" ]]; then
  time_phase "govulncheck source scan" "install govulncheck if needed and scan source packages" run_govulncheck_source_scan
  section "Provider dependency preflight complete"
  exit 0
fi

if [[ "${ONLY_BUILD_NON_FIXTURE:-0}" == "1" ]]; then
  run_build_package_validation
  section "Provider dependency preflight build complete"
  exit 0
fi

if [[ "${ONLY_PROVIDER_COMMAND_TESTS:-0}" == "1" ]]; then
  run_provider_command_test_validation
  section "Provider dependency preflight provider tests complete"
  exit 0
fi

if [[ "${ONLY_CHECK_PROVIDER_COMMAND_TEST_SHARDS:-0}" == "1" ]]; then
  time_phase "Check provider command test shards" "prove provider/cmd test shard union and non-overlap" check_provider_command_test_shards
  section "Provider dependency preflight provider test shard check complete"
  exit 0
fi

run_provider_validation

if [[ "$MODE" == "release" || "${RUN_GOVULNCHECK:-0}" == "1" ]]; then
  time_phase "govulncheck source scan" "install govulncheck if needed and scan source packages" run_govulncheck_source_scan
fi

if [[ "$MODE" == "release" ]]; then
  run_release_validation
fi

section "Provider dependency preflight complete"
