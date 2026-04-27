// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package kubernetes

import (
	"strings"

	"github.com/iancoleman/strcase"
)

func extractClientSetFuncGroupName(group, version string) string {
	v := strings.Title(version)
	if len(group) > 0 {
		return strings.Title(strings.Split(group, ".")[0]) + v
	}
	return "Core" + v
}

func extractClientSetFuncTypeName(kind string) string {
	switch string(kind[len(kind)-1]) {
	case "s":
		return kind + "es"
	case "y":
		return strings.TrimSuffix(kind, "y") + "ies"
	}
	return kind + "s"
}

func extractTfResourceName(kind string) string {
	return "kubernetes_" + strcase.ToSnake(kind)
}
