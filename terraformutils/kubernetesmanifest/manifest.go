// SPDX-License-Identifier: Apache-2.0

package kubernetesmanifest

var serverOwnedMetadataKeys = []string{
	"creationTimestamp",
	"deletionGracePeriodSeconds",
	"deletionTimestamp",
	"generation",
	"managedFields",
	"resourceVersion",
	"selfLink",
	"uid",
}

func ConfigFromObject(object map[string]interface{}) map[string]interface{} {
	manifest := copyMap(object)
	delete(manifest, "status")

	metadata, ok := manifest["metadata"].(map[string]interface{})
	if !ok {
		return manifest
	}
	for _, key := range serverOwnedMetadataKeys {
		delete(metadata, key)
	}
	return manifest
}

func copyMap(value map[string]interface{}) map[string]interface{} {
	copyValue := make(map[string]interface{}, len(value))
	for key, child := range value {
		copyValue[key] = copyInterface(child)
	}
	return copyValue
}

func copyInterface(value interface{}) interface{} {
	switch value := value.(type) {
	case map[string]interface{}:
		return copyMap(value)
	case []interface{}:
		copyValue := make([]interface{}, len(value))
		for i, child := range value {
			copyValue[i] = copyInterface(child)
		}
		return copyValue
	default:
		return value
	}
}
