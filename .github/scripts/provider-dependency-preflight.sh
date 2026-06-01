#!/usr/bin/env bash
set -euo pipefail

MODE="${MODE:-full}"
export GOWORK=off

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

prepare_release_output_cleanup() {
  for path in dist .goreleaser-extra; do
    if git ls-files --error-unmatch "$path" >/dev/null 2>&1; then
      fail "refusing to run release preflight with tracked $path output"
    fi
  done

  release_backup_dir="$(mktemp -d "${TMPDIR:-/tmp}/terraformer-release-preflight.XXXXXX")"
  trap restore_release_outputs EXIT
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

  if ! changed_paths="$(git diff --name-only "${diff_args[@]}" 2>/dev/null)"; then
    warn "could not diff $base_ref...HEAD; falling back to $base_ref HEAD"
    changed_paths="$(git diff --name-only "$base_ref" HEAD)"
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
  gobin="$(go env GOBIN)"
  if [[ -z "$gobin" ]]; then
    gobin="$(go env GOPATH)/bin"
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
    section "$name"
    bash "$script"
    return
  fi

  printf 'Skipping %s; %s is not present/readable.\n' "$name" "$script"
}

run_provider_validation() {
  section "Go module tidy"
  go mod tidy
  git diff --exit-code -- go.mod go.sum

  section "Build all packages"
  go build -v ./...

  section "Test provider and command packages"
  go test ./providers/... ./cmd/... -count=1

  section "Test build and utility packages"
  go test ./build/... ./terraformutils/... ./version -count=1

  section "Vet dependency-sensitive packages"
  go vet ./providers/... ./cmd/... ./build/... ./terraformutils/... ./version

  section "Static diff check"
  git diff --check

  run_compat_script ".github/scripts/terraform-state-compat.sh" "Terraform state compatibility"
  run_compat_script ".github/scripts/terraform-provider-compat.sh" "Terraform provider compatibility"
}

run_govulncheck_source_scan() {
  local batch=()
  local batch_size="${GOVULNCHECK_BATCH_SIZE:-25}"
  local package
  local packages=()
  local scan_level="${GOVULNCHECK_SCAN_LEVEL:-package}"

  ensure_govulncheck

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
    while IFS= read -r package; do
      packages+=("$package")
    done < <(go list ./...)
  fi

  if [[ "${#packages[@]}" -eq 0 ]]; then
    fail "no Go packages found for govulncheck source scan"
  fi

  for package in "${packages[@]}"; do
    batch+=("$package")
    if [[ "${#batch[@]}" -ge "$batch_size" ]]; then
      printf 'Scanning %s package(s) at %s level: %s\n' "${#batch[@]}" "$scan_level" "${batch[*]}"
      govulncheck -scan="$scan_level" "${batch[@]}"
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

  section "GoReleaser check"
  goreleaser check

  section "GoReleaser snapshot preflight"
  prepare_release_output_cleanup
  goreleaser release --snapshot --clean --skip=publish --timeout=3h --parallelism=1
}

if [[ "$MODE" == "quick" ]]; then
  section "Detect dependency-sensitive changes"
  base_ref="$(resolve_base_ref)"
  printf 'Using base ref: %s\n' "$base_ref"
  if ! has_dependency_sensitive_changes "$base_ref"; then
    printf 'No dependency-sensitive changes detected; skipping provider dependency preflight.\n'
    exit 0
  fi
  printf 'Dependency-sensitive changes detected; running provider dependency preflight.\n'
fi

if [[ "${ONLY_GOVULNCHECK:-0}" == "1" ]]; then
  run_govulncheck_source_scan
  section "Provider dependency preflight complete"
  exit 0
fi

run_provider_validation

if [[ "$MODE" == "release" || "${RUN_GOVULNCHECK:-0}" == "1" ]]; then
  run_govulncheck_source_scan
fi

if [[ "$MODE" == "release" ]]; then
  run_release_validation
fi

section "Provider dependency preflight complete"
