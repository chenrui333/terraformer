#!/usr/bin/env bash
set -euo pipefail

SCRIPT_STARTED_AT="$(date +%s)"
TEST_OUTPUT=""

markdown_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//|/\\|}"
  value="${value//$'\n'/ }"
  printf '%s\n' "$value"
}

write_step_summary() {
  local exit_code="$1"
  local total_duration="$2"
  local package_rows

  [[ -n "${GITHUB_STEP_SUMMARY:-}" ]] || return 0

  package_rows="$(
    awk '
      ($1 == "ok" || $1 == "FAIL") && $3 ~ /^[0-9.]+s$/ {
        duration=$3
        sub(/s$/, "", duration)
        printf "%s\t%s\n", $2, duration
      }
    ' "$TEST_OUTPUT" | sort -k2,2nr | awk 'NR <= 20 { print }'
  )"

  {
    printf '## PR Core Test Timing\n\n'
    printf -- '- Result: %s\n' "$([[ "$exit_code" -eq 0 ]] && printf success || printf failed)"
    printf -- '- Total duration seconds: %s\n' "$total_duration"
    printf -- '- Command: go test ./build/... ./terraformutils/... ./version ./cmd ./providers ./providers/aws ./providers/gcp ./providers/azure -count=1\n\n'
    printf '| Package | Duration seconds |\n'
    printf '| --- | ---: |\n'
    if [[ -n "$package_rows" ]]; then
      while IFS=$'\t' read -r package seconds; do
        printf '| %s | %s |\n' "$(markdown_escape "$package")" "$seconds"
      done <<<"$package_rows"
    else
      printf '| _No package timing rows found_ | 0 |\n'
    fi
  } >>"$GITHUB_STEP_SUMMARY"
}

trap 'rm -f "$TEST_OUTPUT"' EXIT

TEST_OUTPUT="$(mktemp "${TMPDIR:-/tmp}/terraformer-pr-core-test.XXXXXX")"

packages=(
  ./build/...
  ./terraformutils/...
  ./version
  ./cmd
  ./providers
  ./providers/aws
  ./providers/gcp
  ./providers/azure
)

set +e
go test "${packages[@]}" -count=1 2>&1 | tee "$TEST_OUTPUT"
test_status="${PIPESTATUS[0]}"
set -e

total_duration="$(($(date +%s) - SCRIPT_STARTED_AT))"
if [[ "$test_status" -eq 0 ]]; then
  status="success"
else
  status="failed"
fi

printf 'pr-core-test: timing phase=%q status=%s duration=%ss notes=%q\n' \
  "Test core packages" "$status" "$total_duration" "go test selected PR packages"
write_step_summary "$test_status" "$total_duration"

exit "$test_status"
