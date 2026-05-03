// SPDX-License-Identifier: Apache-2.0

package opal

import (
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/secure/precis"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func opalRequiredString(resourceType, field, value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s resource is missing %s", resourceType, field)
	}
	return value, nil
}

func opalRequiredStringPtr(resourceType, field string, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", fmt.Errorf("%s resource is missing %s", resourceType, field)
	}
	return *value, nil
}

func opalResourceDisplayName(name *string, fallback string) string {
	if name != nil && *name != "" {
		return *name
	}
	return fallback
}

func opalUniqueResourceName(name string, countByName map[string]int) string {
	normalizedName := normalizeResourceName(name)
	if count, ok := countByName[normalizedName]; ok {
		countByName[normalizedName] = count + 1
		return normalizeResourceName(fmt.Sprintf("%s_%d", name, count+1))
	}
	countByName[normalizedName] = 1
	return normalizedName
}

func normalizeResourceName(s string) string {
	normalize := precis.NewIdentifier(
		precis.AdditionalMapping(func() transform.Transformer {
			return transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool { //nolint
				return unicode.Is(unicode.Mn, r)
			}))
		}),
		precis.Norm(norm.NFC),
	)
	r := strings.NewReplacer(" ", "_",
		"!", "_",
		"\"", "_",
		"#", "_",
		"%", "_",
		"&", "_",
		"'", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"*", "_",
		"+", "_",
		",", "_",
		"-", "_",
		".", "_",
		"/", "slash",
		"|", "_",
		"\\", "_",
		":", "_",
		";", "_",
		">", "_",
		"=", "_",
		"<", "_",
		"?", "_",
		"[", "_",
		"]", "_",
		"^", "_",
		"`", "_",
		"~", "_",
		"$", "_",
		"@", "_at_")
	normalizedString, _ := normalize.String(r.Replace(strings.ToLower(s)))
	return normalizedString
}
