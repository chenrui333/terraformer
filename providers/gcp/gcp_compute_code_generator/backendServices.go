// SPDX-License-Identifier: Apache-2.0

package main

type backendServices struct {
	basicGCPResource
}

func (b backendServices) ifNeedRegion() bool {
	return false
}
