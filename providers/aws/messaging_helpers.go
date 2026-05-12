// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strconv"
	"strings"
)

func resourceNameWithLengthPrefixes(parts ...string) string {
	var name strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if name.Len() > 0 {
			name.WriteString("_")
		}
		name.WriteString(strconv.Itoa(len(part)))
		name.WriteString("_")
		name.WriteString(part)
	}
	return name.String()
}

func stringSliceToInterfaceSlice(values []string) []interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
