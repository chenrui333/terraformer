// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package okta

import (
	"fmt"
	"math/rand"
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

func normalizeResourceNameWithRandom(s string) string {
	specialChars := `-<>()*#{}[]|@_ .%'",&`
	for _, c := range specialChars {
		s = strings.ReplaceAll(s, string(c), "_")
	}
	s = regexp.MustCompile(`^[^a-zA-Z_]+`).ReplaceAllLiteralString(s, "")
	s = strings.TrimSuffix(s, "`_")
	randString := RandStringBytes(4)
	return fmt.Sprintf("%s_%s", strings.ToLower(s), randString)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// escapeDollar modifies ${ into $${ recursively
func escapeDollar(item map[string]interface{}) map[string]interface{} {
	for k, f := range item {
		switch v := f.(type) {
		case string:
			item[k] = strings.ReplaceAll(v, "${", "$${")
		case map[string]interface{}:
			item[k] = escapeDollar(v)
		case []interface{}:
			for i, s := range v {
				if str, ok := s.(string); ok {
					v[i] = strings.ReplaceAll(str, "${", "$${")
				}
			}
			item[k] = v
		}
	}
	return item
}
