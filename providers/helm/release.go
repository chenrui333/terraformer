// SPDX-License-Identifier: Apache-2.0

package helm

import "github.com/chenrui333/terraformer/terraformutils"

type ReleaseGenerator struct {
	terraformutils.Service
}

func (g *ReleaseGenerator) InitResources() error {
	return ErrReleaseImportNotImplemented
}
