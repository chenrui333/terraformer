// SPDX-License-Identifier: Apache-2.0

package main

type gcpResourceRenderable interface {
	getTerraformName() string
	getAdditionalFields() map[string]string
	getAllowEmptyValues() []string
	ifNeedRegion() bool
	ifNeedZone(zoneInParameters bool) bool
	ifIDWithZone(zoneInParameters bool) bool
	getAdditionalFieldsForRefresh() map[string]string
}

type basicGCPResource struct {
	terraformName              string
	allowEmptyValues           []string
	additionalFields           map[string]string
	additionalFieldsForRefresh map[string]string
}

func (b basicGCPResource) getTerraformName() string {
	return b.terraformName
}

func (b basicGCPResource) getAdditionalFields() map[string]string {
	return b.additionalFields
}

func (b basicGCPResource) getAdditionalFieldsForRefresh() map[string]string {
	return b.additionalFieldsForRefresh
}

func (b basicGCPResource) getAllowEmptyValues() []string {
	return b.allowEmptyValues
}
func (b basicGCPResource) ifNeedRegion() bool {
	return true
}

func (b basicGCPResource) ifNeedZone(zoneInParameters bool) bool {
	return zoneInParameters
}

func (b basicGCPResource) ifIDWithZone(zoneInParameters bool) bool {
	return zoneInParameters
}
