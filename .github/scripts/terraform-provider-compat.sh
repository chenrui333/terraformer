#!/usr/bin/env bash
set -euo pipefail

export TERRAFORMER_PROVIDER_COMPAT_TEST=1
GOWORK="${GOWORK:-off}" go test ./cmd -run '^TestProviderRegistryCompatibility$' -count=1 -v
