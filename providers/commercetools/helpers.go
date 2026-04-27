// SPDX-License-Identifier: Apache-2.0

package commercetools

import (
	"regexp"
	"strings"
)

// Making resource's name less ugly
func normalizeResourceName(s string) string {
	specialChars := `<>()*#{}[]|@_ .%'",&`
	for _, c := range specialChars {
		s = strings.ReplaceAll(s, string(c), "-")
	}

	s = regexp.MustCompile(`^[^a-zA-Z_]+`).ReplaceAllLiteralString(s, "")
	s = strings.TrimSuffix(s, "-")

	return strings.ToLower(s)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
