#!/usr/bin/env bash

set -euo pipefail

# Keep this default list aligned with Terraform minor lines supported by generated state.
minors=${TERRAFORM_COMPAT_MINORS:-"1.9 1.10 1.11 1.12 1.13 1.14"}
work_dir=${RUNNER_TEMP:-${TMPDIR:-/tmp}}/terraformer-state-compat

rm -rf "$work_dir"
mkdir -p "$work_dir/bin" "$work_dir/downloads"
trap 'rm -rf "$work_dir"' EXIT

case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
  darwin) os=darwin ;;
  linux) os=linux ;;
  *) echo "unsupported os: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  arm64|aarch64) arch=arm64 ;;
  x86_64) arch=amd64 ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

index_path="$work_dir/terraform-index.json"
curl -fsSL --retry 3 https://releases.hashicorp.com/terraform/index.json -o "$index_path"

for minor in $minors; do
  version=$(
    jq -r --arg prefix "$minor." '
      .versions
      | keys[]
      | select(startswith($prefix))
      | select(test("^([0-9]+\\.){2}[0-9]+$"))
      | . as $version
      | split(".") as $parts
      | [$parts[2] | tonumber, $version]
      | @tsv
    ' "$index_path" | sort -n | tail -n 1 | cut -f 2
  )

  if [[ -z "$version" ]]; then
    echo "could not resolve latest Terraform $minor.x release" >&2
    exit 1
  fi

  terraform_dir="$work_dir/bin/terraform-$version"
  terraform_zip="$work_dir/downloads/terraform_${version}_${os}_${arch}.zip"
  terraform_url="https://releases.hashicorp.com/terraform/${version}/terraform_${version}_${os}_${arch}.zip"

  echo "::group::Terraform $version"
  mkdir -p "$terraform_dir"
  curl -fsSL --retry 3 "$terraform_url" -o "$terraform_zip"
  unzip -q "$terraform_zip" -d "$terraform_dir"
  terraform_version_output=$("$terraform_dir/terraform" version)
  printf '%s\n' "${terraform_version_output%%$'\n'*}"

  PATH="$terraform_dir:$PATH" GOWORK=off go test ./terraformutils \
    -run 'TestPrintTfState(WritesV4ProviderSourceState|CanBeListedByTerraformCLI)$' \
    -count=1 \
    -v
  echo "::endgroup::"
done
