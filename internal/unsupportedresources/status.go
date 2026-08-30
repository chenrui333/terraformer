// SPDX-License-Identifier: Apache-2.0

// Package unsupportedresources defines the repository-wide unsupported-resource
// metadata status vocabulary.
package unsupportedresources

import "slices"

var canonicalStatuses = [...]string{
	"action-style",
	"cloudflare-managed",
	"deferred",
	"not-importable",
	"policy-skip",
	"request-style",
	"runtime-data",
	"runtime-generated",
	"secret-required",
	"unsupported",
}

// IsValidStatus reports whether status exactly matches a canonical status.
func IsValidStatus(status string) bool {
	_, found := slices.BinarySearch(canonicalStatuses[:], status)
	return found
}

// Statuses returns the sorted canonical status vocabulary.
func Statuses() []string {
	return slices.Clone(canonicalStatuses[:])
}
