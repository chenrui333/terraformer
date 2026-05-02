// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/iancoleman/strcase"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

const (
	labelsServiceName       = "labels"
	labelsTerraformType     = "kubernetes_labels"
	labelsAllowEmptyPattern = `^labels\.`

	annotationsServiceName       = "annotations"
	annotationsTerraformType     = "kubernetes_annotations"
	annotationsAllowEmptyPattern = `^annotations\.`

	metadataPatchListTimeout = 30 * time.Second
)

type MetadataPatch struct {
	KubernetesService
	TerraformType     string
	AttributeName     string
	AllowEmptyPattern string
}

func (m *MetadataPatch) InitResources() error {
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

	return m.initResources(client, lists)
}

func kubernetesPreferredResources(dc discovery.DiscoveryInterface) ([]*metav1.APIResourceList, error) {
	lists, err := dc.ServerPreferredResources()
	if err != nil {
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, err
		}
		log.Printf("kubernetes: discovery skipped unavailable API groups: %v", err)
	}
	return lists, nil
}

func (m *MetadataPatch) initResources(client dynamic.Interface, lists []*metav1.APIResourceList) error {
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}

		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if !metadataPatchSupportsResource(resource) {
				continue
			}
			if err := m.initResourceList(client, gv, resource); err != nil {
				log.Printf("kubernetes: metadata patch skipped %s/%s: %v", list.GroupVersion, resource.Name, err)
				continue
			}
		}
	}
	return nil
}

func (m *MetadataPatch) initResourceList(client dynamic.Interface, gv schema.GroupVersion, resource metav1.APIResource) error {
	resourceClient := client.Resource(schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource.Name,
	})
	listClient := dynamic.ResourceInterface(resourceClient)
	if resource.Namespaced {
		listClient = resourceClient.Namespace(metav1.NamespaceAll)
	}

	ctx, cancel := context.WithTimeout(context.Background(), metadataPatchListTimeout)
	defer cancel()

	results, err := listClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	apiVersion := metadataPatchAPIVersion(gv)
	terraformType := m.terraformType()
	for i := range results.Items {
		item := results.Items[i]
		if len(item.GetOwnerReferences()) > 0 {
			continue
		}
		values := metadataPatchValues(item, m.AttributeName)
		if len(values) == 0 {
			continue
		}

		importID := metadataPatchID(apiVersion, resource.Kind, item.GetNamespace(), item.GetName(), resource.Namespaced)
		resourceName := metadataPatchResourceName(m.AttributeName, apiVersion, resource.Kind, item.GetNamespace(), item.GetName(), resource.Namespaced)
		m.Resources = append(m.Resources, terraformutils.NewResource(
			importID,
			resourceName,
			terraformType,
			"kubernetes",
			metadataPatchAttributes(item, apiVersion, resource.Kind, m.AttributeName, values, resource.Namespaced),
			[]string{m.AllowEmptyPattern},
			map[string]interface{}{},
		))
	}
	return nil
}

func metadataPatchSupportsResource(resource metav1.APIResource) bool {
	if resource.Kind == "" || strings.Contains(resource.Name, "/") {
		return false
	}
	return metadataPatchHasVerbs(resource, "get", "list", "patch")
}

func metadataPatchHasVerbs(resource metav1.APIResource, required ...string) bool {
	verbs := map[string]struct{}{}
	for _, verb := range resource.Verbs {
		verbs[verb] = struct{}{}
	}
	for _, verb := range required {
		if _, ok := verbs[verb]; !ok {
			return false
		}
	}
	return true
}

func (m *MetadataPatch) terraformType() string {
	if m.TerraformType != "" {
		return m.TerraformType
	}
	switch m.AttributeName {
	case "labels":
		return labelsTerraformType
	case "annotations":
		return annotationsTerraformType
	default:
		return ""
	}
}

func metadataPatchValues(item unstructured.Unstructured, attributeName string) map[string]string {
	switch attributeName {
	case "labels":
		return item.GetLabels()
	case "annotations":
		return item.GetAnnotations()
	default:
		return nil
	}
}

func metadataPatchAttributes(item unstructured.Unstructured, apiVersion, kind, attributeName string, values map[string]string, namespaced bool) map[string]string {
	attributes := map[string]string{
		"id":                 metadataPatchID(apiVersion, kind, item.GetNamespace(), item.GetName(), namespaced),
		"api_version":        apiVersion,
		"kind":               kind,
		"metadata.#":         "1",
		"metadata.0.name":    item.GetName(),
		attributeName + ".%": strconv.Itoa(len(values)),
	}
	if namespaced {
		attributes["metadata.0.namespace"] = item.GetNamespace()
	}
	for key, value := range values {
		attributes[attributeName+"."+key] = value
	}
	return attributes
}

func metadataPatchID(apiVersion, kind, namespace, name string, namespaced bool) string {
	parts := []string{
		"apiVersion=" + apiVersion,
		"kind=" + kind,
	}
	if namespaced {
		parts = append(parts, "namespace="+namespace)
	}
	parts = append(parts, "name="+name)
	return strings.Join(parts, ",")
}

func metadataPatchObjectKey(kind, namespace, name string, namespaced bool) string {
	parts := []string{"kind=" + kind}
	if namespaced {
		parts = append(parts, "namespace="+namespace)
	}
	parts = append(parts, "name="+name)
	return strings.Join(parts, ",")
}

func metadataPatchObjectKeyAndAPIVersionFromID(id string) (string, string, bool) {
	values := map[string]string{}
	for _, part := range strings.Split(id, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return "", "", false
		}
		values[key] = value
	}

	apiVersion := values["apiVersion"]
	kind := values["kind"]
	name := values["name"]
	if apiVersion == "" || kind == "" || name == "" {
		return "", "", false
	}
	namespace, namespaced := values["namespace"]
	return metadataPatchObjectKey(kind, namespace, name, namespaced), apiVersion, true
}

func metadataPatchResourceName(attributeName, apiVersion, kind, namespace, name string, namespaced bool) string {
	parts := []string{attributeName, apiVersion, kind}
	if namespaced {
		parts = append(parts, namespace)
	}
	parts = append(parts, name)
	return strings.Join(parts, "/")
}

func metadataPatchAPIVersion(gv schema.GroupVersion) string {
	if gv.Group == "" {
		return gv.Version
	}
	return gv.Group + "/" + gv.Version
}

func metadataPatchTargetIDs(resource terraformutils.Resource) []string {
	if resource.InstanceInfo == nil || resource.InstanceState == nil {
		return nil
	}
	if resource.InstanceInfo.Type == manifestTerraformResourceName && strings.HasPrefix(resource.InstanceState.ID, "apiVersion=") {
		return []string{resource.InstanceState.ID}
	}

	name, namespace, namespaced, ok := metadataPatchResourceObject(resource)
	if !ok {
		return nil
	}

	ids := []string{}
	for _, resourceID := range metadataPatchResourceKindsForTerraformType(resource.InstanceInfo.Type) {
		apiVersion := metadataPatchAPIVersion(schema.GroupVersion{Group: resourceID.group, Version: resourceID.version})
		ids = append(ids, metadataPatchID(apiVersion, resourceID.kind, namespace, name, namespaced))
	}
	return ids
}

func metadataPatchFallbackTargetKeys(resource terraformutils.Resource) []string {
	if resource.InstanceInfo == nil || resource.InstanceState == nil {
		return nil
	}
	if len(metadataPatchResourceKindsForTerraformType(resource.InstanceInfo.Type)) != 0 {
		return nil
	}

	// selectTerraformResourceName can choose extractTfResourceName(kind) for
	// typed resources that have no explicit preferred mapping; by this point the
	// discovered group/version is no longer available, so match on object shape.
	kind, ok := metadataPatchFallbackKindForTerraformType(resource.InstanceInfo.Type)
	if !ok {
		return nil
	}
	name, namespace, namespaced, ok := metadataPatchResourceObject(resource)
	if !ok {
		return nil
	}
	return []string{metadataPatchObjectKey(kind, namespace, name, namespaced)}
}

func metadataPatchResourceObject(resource terraformutils.Resource) (string, string, bool, bool) {
	name := resource.InstanceState.Attributes["metadata.0.name"]
	if name != "" {
		namespace, namespaced := resource.InstanceState.Attributes["metadata.0.namespace"]
		return name, namespace, namespaced, true
	}
	return metadataPatchResourceObjectFromImportID(resource.InstanceState.ID)
}

func metadataPatchResourceObjectFromImportID(id string) (string, string, bool, bool) {
	if id == "" || strings.HasPrefix(id, "apiVersion=") {
		return "", "", false, false
	}
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", false, false
		}
		return parts[0], "", false, true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", false, false
		}
		return parts[1], parts[0], true, true
	default:
		return "", "", false, false
	}
}

func metadataPatchResourceKindsForTerraformType(terraformType string) []kubernetesResourceID {
	seen := map[kubernetesResourceID]struct{}{}
	resourceIDs := []kubernetesResourceID{}
	for resourceID := range preferredTerraformResourceNames {
		for _, candidate := range terraformResourceNameCandidates(resourceID.group, resourceID.version, resourceID.kind) {
			if candidate != terraformType {
				continue
			}
			if _, ok := seen[resourceID]; ok {
				continue
			}
			seen[resourceID] = struct{}{}
			resourceIDs = append(resourceIDs, resourceID)
		}
	}

	switch terraformType {
	case "kubernetes_default_service_account", "kubernetes_default_service_account_v1":
		serviceAccount := kubernetesResourceID{version: "v1", kind: "ServiceAccount"}
		if _, ok := seen[serviceAccount]; !ok {
			resourceIDs = append(resourceIDs, serviceAccount)
		}
	}
	return resourceIDs
}

func metadataPatchFallbackKindForTerraformType(terraformType string) (string, bool) {
	const prefix = "kubernetes_"
	if !strings.HasPrefix(terraformType, prefix) {
		return "", false
	}
	switch terraformType {
	case labelsTerraformType, annotationsTerraformType, envTerraformType, manifestTerraformResourceName:
		return "", false
	}

	kind := strcase.ToCamel(strings.TrimPrefix(terraformType, prefix))
	if kind == "" || extractTfResourceName(kind) != terraformType {
		return "", false
	}
	return kind, true
}
