// SPDX-License-Identifier: Apache-2.0

package opal

import (
	"strings"
	"unicode"

	"golang.org/x/text/secure/precis"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

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
