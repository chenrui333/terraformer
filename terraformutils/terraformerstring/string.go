// SPDX-License-Identifier: Apache-2.0

package terraformerstring

func ContainsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
