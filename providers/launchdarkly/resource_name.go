// SPDX-License-Identifier: Apache-2.0

package launchdarkly

func resourceName(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}
