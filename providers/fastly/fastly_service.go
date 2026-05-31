// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

type FastlyService struct { //nolint
	terraformutils.Service
}

func fastlyStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func fastlyIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
