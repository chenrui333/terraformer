// SPDX-License-Identifier: Apache-2.0

package terraformutils

func ConnectServices(importResources map[string][]Resource, isServicePath bool, resourceConnections map[string]map[string][]string) map[string][]Resource {
	for resource, connection := range resourceConnections {
		if _, exist := importResources[resource]; exist {
			for k, connectionPairs := range connection {
				if len(connectionPairs)%2 == 1 {
					continue
				}
				if cc, ok := importResources[k]; ok {
					for i := 0; i < len(connectionPairs)/2; i++ {
						connectionPair := []string{connectionPairs[i*2], connectionPairs[i*2+1]}
						for _, ccc := range cc {
							if !isServicePath {
								mapResource(importResources, resource, connectionPair, ccc, "local")
							} else {
								mapResource(importResources, resource, connectionPair, ccc, k)
							}
						}
					}
				}
			}
		}
	}
	return importResources
}

func mapResource(importResources map[string][]Resource, resource string, connectionPair []string, resourceToMap Resource, k string) {
	for i := range importResources[resource] {
		key := connectionPair[1]
		if connectionPair[1] == "self_link" || connectionPair[1] == "id" {
			key = resourceToMap.GetIDKey()
		}
		mappingResourceAttr := WalkAndGet(key, resourceToMap.InstanceState.Attributes)
		keyValue := resourceToMap.InstanceInfo.Type + "_" + resourceToMap.ResourceName + "_" + key
		linkValue := "${data.terraform_remote_state." + k + ".outputs." + keyValue + "}"

		if len(mappingResourceAttr) == 1 {
			resourceIdentifier := mappingResourceAttr[0].(string)
			WalkAndOverride(connectionPair[0], resourceIdentifier, linkValue, importResources[resource][i].Item)
		}
	}
}
