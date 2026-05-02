// SPDX-License-Identifier: Apache-2.0

package launchdarkly

func resourceName(name, fallback string) string {
	if name != "" {
		return name
	}
	return fallback
}

func resourceNameWithID(name, id string) string {
	if name == "" || id == "" {
		return resourceName(name, id)
	}
	return name + "-" + id
}
