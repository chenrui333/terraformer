// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"errors"
	"testing"
)

func TestReleaseGeneratorInitResourcesNotImplemented(t *testing.T) {
	generator := &ReleaseGenerator{}
	if err := generator.InitResources(); !errors.Is(err, ErrReleaseImportNotImplemented) {
		t.Fatalf("InitResources() error = %v, want %v", err, ErrReleaseImportNotImplemented)
	}
}
