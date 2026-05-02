// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

const (
	envServiceName       = "env"
	envTerraformType     = "kubernetes_env"
	envAllowEmptyPattern = "^env\\.[0-9]+\\.value$"
	envListTimeout       = 30 * time.Second
)

var envSupportedKinds = map[string]struct{}{
	"CronJob":               {},
	"DaemonSet":             {},
	"Deployment":            {},
	"ReplicaSet":            {},
	"ReplicationController": {},
	"StatefulSet":           {},
}

type Env struct {
	KubernetesService
	TerraformType string
}

type envContainer struct {
	Name          string
	InitContainer bool
	Envs          []map[string]interface{}
}

func (e *Env) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return err
	}
	lists, err := kubernetesPreferredResources(dc)
	if err != nil {
		return err
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	return e.initResources(client, lists)
}

func (e *Env) initResources(client dynamic.Interface, lists []*metav1.APIResourceList) error {
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}

		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if !envSupportsResource(resource) {
				continue
			}
			if err := e.initResourceList(client, gv, resource); err != nil {
				log.Printf("kubernetes: env skipped %s/%s: %v", list.GroupVersion, resource.Name, err)
				continue
			}
		}
	}
	return nil
}

func (e *Env) initResourceList(client dynamic.Interface, gv schema.GroupVersion, resource metav1.APIResource) error {
	resourceClient := client.Resource(schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	})
	listClient := dynamic.ResourceInterface(resourceClient)
	if resource.Namespaced {
		listClient = resourceClient.Namespace(metav1.NamespaceAll)
	}

	ctx, cancel := context.WithTimeout(context.Background(), envListTimeout)
	defer cancel()

	results, err := listClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	apiVersion := metadataPatchAPIVersion(gv)
	terraformType := e.terraformType()
	for i := range results.Items {
		item := results.Items[i]
		if len(item.GetOwnerReferences()) > 0 {
			continue
		}
		for _, container := range envContainers(item, resource.Kind) {
			for _, env := range container.Envs {
				envName, _ := env["name"].(string)
				importID := metadataPatchID(apiVersion, resource.Kind, item.GetNamespace(), item.GetName(), resource.Namespaced)
				resourceName := envResourceName(apiVersion, resource.Kind, item.GetNamespace(), item.GetName(), container, envName, resource.Namespaced)
				e.Resources = append(e.Resources, terraformutils.NewResource(
					importID,
					resourceName,
					terraformType,
					"kubernetes",
					envAttributes(item, apiVersion, resource.Kind, container, env, resource.Namespaced),
					[]string{envAllowEmptyPattern},
					map[string]interface{}{},
				))
			}
		}
	}
	return nil
}

func (e *Env) terraformType() string {
	if e.TerraformType != "" {
		return e.TerraformType
	}
	return envTerraformType
}

func envSupportsResource(resource metav1.APIResource) bool {
	if resource.Kind == "" || strings.Contains(resource.Name, "/") || !resource.Namespaced {
		return false
	}
	if !envSupportsKind(resource.Kind) {
		return false
	}
	return metadataPatchHasVerbs(resource, "get", "list", "patch")
}

func envSupportsKind(kind string) bool {
	_, ok := envSupportedKinds[kind]
	return ok
}

func envContainers(item unstructured.Unstructured, kind string) []envContainer {
	specs := []struct {
		init bool
		path []string
	}{
		{path: []string{"spec", "template", "spec", "containers"}},
		{init: true, path: []string{"spec", "template", "spec", "initContainers"}},
	}
	if kind == "CronJob" {
		specs = []struct {
			init bool
			path []string
		}{
			{path: []string{"spec", "jobTemplate", "spec", "template", "spec", "containers"}},
			{init: true, path: []string{"spec", "jobTemplate", "spec", "template", "spec", "initContainers"}},
		}
	}

	containers := []envContainer{}
	for _, spec := range specs {
		values, found, _ := unstructured.NestedSlice(item.Object, spec.path...)
		if !found {
			continue
		}
		for _, value := range values {
			container, ok := value.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := container["name"].(string)
			if name == "" {
				continue
			}
			envValues, ok := container["env"].([]interface{})
			if !ok || len(envValues) == 0 {
				continue
			}
			envs := envEntries(envValues)
			if len(envs) == 0 {
				continue
			}
			containers = append(containers, envContainer{
				Name:          name,
				InitContainer: spec.init,
				Envs:          envs,
			})
		}
	}
	return containers
}

func envEntries(values []interface{}) []map[string]interface{} {
	countsByName := map[string]int{}
	entries := make([]map[string]interface{}, 0, len(values))
	for _, value := range values {
		env, ok := value.(map[string]interface{})
		if !ok {
			entries = append(entries, nil)
			continue
		}
		name, _ := env["name"].(string)
		if name == "" {
			entries = append(entries, nil)
			continue
		}
		entries = append(entries, env)
		countsByName[name]++
	}

	envs := []map[string]interface{}{}
	for _, env := range entries {
		if env == nil {
			continue
		}
		name, _ := env["name"].(string)
		// kubernetes_env refresh filters by env name, so duplicate live names
		// cannot be imported without reading duplicate blocks back into state.
		if countsByName[name] != 1 {
			continue
		}
		envs = append(envs, env)
	}
	return envs
}

func envAttributes(item unstructured.Unstructured, apiVersion, kind string, container envContainer, env map[string]interface{}, namespaced bool) map[string]string {
	envName, _ := env["name"].(string)
	attributes := map[string]string{
		"id":              metadataPatchID(apiVersion, kind, item.GetNamespace(), item.GetName(), namespaced),
		"api_version":     apiVersion,
		"kind":            kind,
		"field_manager":   envFieldManager(apiVersion, kind, item.GetNamespace(), item.GetName(), container, envName, namespaced),
		"metadata.#":      "1",
		"metadata.0.name": item.GetName(),
		"env.#":           "1",
	}
	if namespaced {
		attributes["metadata.0.namespace"] = item.GetNamespace()
	}
	if container.InitContainer {
		attributes["init_container"] = container.Name
	} else {
		attributes["container"] = container.Name
	}
	envEntryAttributes(attributes, 0, env)
	return attributes
}

func envEntryAttributes(attributes map[string]string, index int, env map[string]interface{}) {
	prefix := fmt.Sprintf("env.%d", index)
	attributes[prefix+".name"] = fmt.Sprint(env["name"])
	if valueFrom, ok := env["valueFrom"].(map[string]interface{}); ok && len(valueFrom) > 0 {
		attributes[prefix+".value_from.#"] = "1"
		envValueFromAttributes(attributes, prefix+".value_from.0", valueFrom)
		return
	}
	attributes[prefix+".value_from.#"] = "0"
	value, _ := env["value"].(string)
	attributes[prefix+".value"] = value
}

func envValueFromAttributes(attributes map[string]string, prefix string, valueFrom map[string]interface{}) {
	for _, child := range []string{"config_map_key_ref", "field_ref", "resource_field_ref", "secret_key_ref"} {
		attributes[prefix+"."+child+".#"] = "0"
	}
	envRefAttributes(attributes, prefix+".config_map_key_ref", valueFrom["configMapKeyRef"], map[string]string{
		"key":      "key",
		"name":     "name",
		"optional": "optional",
	})
	envRefAttributes(attributes, prefix+".field_ref", valueFrom["fieldRef"], map[string]string{
		"apiVersion": "api_version",
		"fieldPath":  "field_path",
	})
	envRefAttributes(attributes, prefix+".resource_field_ref", valueFrom["resourceFieldRef"], map[string]string{
		"containerName": "container_name",
		"divisor":       "divisor",
		"resource":      "resource",
	})
	envRefAttributes(attributes, prefix+".secret_key_ref", valueFrom["secretKeyRef"], map[string]string{
		"key":      "key",
		"name":     "name",
		"optional": "optional",
	})
}

func envRefAttributes(attributes map[string]string, prefix string, raw interface{}, fields map[string]string) {
	ref, ok := raw.(map[string]interface{})
	if !ok || len(ref) == 0 {
		return
	}
	attributes[prefix+".#"] = "1"
	for source, target := range fields {
		value, ok := ref[source]
		if !ok || value == nil {
			continue
		}
		attributes[prefix+".0."+target] = fmt.Sprint(value)
	}
}

func envResourceName(apiVersion, kind, namespace, name string, container envContainer, envName string, namespaced bool) string {
	parts := []string{"env", apiVersion, kind}
	if namespaced {
		parts = append(parts, namespace)
	}
	parts = append(parts, name)
	if container.InitContainer {
		parts = append(parts, "init_container")
	} else {
		parts = append(parts, "container")
	}
	parts = append(parts, container.Name)
	parts = append(parts, envName)
	return strings.Join(parts, "/")
}

func envFieldManager(apiVersion, kind, namespace, name string, container envContainer, envName string, namespaced bool) string {
	id := metadataPatchID(apiVersion, kind, namespace, name, namespaced)
	containerType := "container"
	if container.InitContainer {
		containerType = "init_container"
	}
	checksum := crc32.ChecksumIEEE([]byte(strings.Join([]string{id, containerType, container.Name, envName}, "/")))
	return fmt.Sprintf("terraformer-env-%08x", checksum)
}

func envTargetIDs(resource terraformutils.Resource) []string {
	if resource.InstanceInfo == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceInfo.Type == manifestTerraformResourceName && strings.HasPrefix(resource.InstanceState.ID, "apiVersion=") {
		if kind, ok := envKindFromID(resource.InstanceState.ID); ok && envSupportsKind(kind) {
			return []string{resource.InstanceState.ID}
		}
		return nil
	}

	name, namespace, namespaced, ok := metadataPatchResourceObject(resource)
	if !ok {
		return nil
	}
	ids := []string{}
	for _, resourceID := range metadataPatchResourceKindsForTerraformType(resource.InstanceInfo.Type) {
		if !envSupportsKind(resourceID.kind) {
			continue
		}
		apiVersion := metadataPatchAPIVersion(schema.GroupVersion{Group: resourceID.group, Version: resourceID.version})
		ids = append(ids, metadataPatchID(apiVersion, resourceID.kind, namespace, name, namespaced))
	}
	return ids
}

func envKindFromID(id string) (string, bool) {
	for _, part := range strings.Split(id, ",") {
		key, value, ok := strings.Cut(part, "=")
		if ok && key == "kind" && value != "" {
			return value, true
		}
	}
	return "", false
}
