// SPDX-License-Identifier: Apache-2.0

package newrelic

import (
	"regexp"
	"strings"
)

func removeDuplicate(s string) string {
	if len(s) < 2 {
		return s
	}

	src := []byte(s)
	dest := make([]byte, len(src))
	dest[0] = src[0]

	j := 0
	for i := 1; i < len(s); i++ {
		if dest[j] != src[i] {
			j++
			dest[j] = src[i]
		}
	}

	return string(dest[:j+1])
}

// Making resource's name less ugly
func normalizeResourceName(s string) string {
	specialChars := `<>()*#{}[]|@_ .%'",&`
	for _, c := range specialChars {
		s = strings.ReplaceAll(s, string(c), "-")
	}

	s = regexp.MustCompile(`^[^a-zA-Z_]+`).ReplaceAllLiteralString(s, "")
	s = strings.TrimSuffix(s, "-")
	s = removeDuplicate(s)

	return strings.ToLower(s)
}
