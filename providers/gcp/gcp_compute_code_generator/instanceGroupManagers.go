// SPDX-License-Identifier: Apache-2.0

package main

type instanceGroupManagers struct {
	basicGCPResource
}

func (b instanceGroupManagers) ifNeedZone(_ bool) bool {
	return true
}

func (b instanceGroupManagers) ifIDWithZone(_ bool) bool {
	return false
}
func (b instanceGroupManagers) ifNeedRegion() bool {
	return false
}
