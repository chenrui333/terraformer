// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/chenrui333/terraformer/terraformutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Kind struct {
	KubernetesService
	Name             string
	ResourceName     string
	Group            string
	Version          string
	Namespaced       bool
	TerraformType    string
	UseDynamicClient bool
}

// Generate TerraformResources from Kubernetes API,
// from each kubernetes object 1 TerraformResource.
// Use UID as the resource IDs.
func (k *Kind) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	if k.UseDynamicClient {
		client, err := dynamic.NewForConfig(config)
		if err != nil {
			return err
		}
		return k.initDynamicResources(client)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return k.initTypedResources(clientset)
}

func (k *Kind) initTypedResources(clientset kubernetes.Interface) error {
	groupMethod := reflect.ValueOf(clientset).MethodByName(extractClientSetFuncGroupName(k.Group, k.Version))
	if !groupMethod.IsValid() {
		return fmt.Errorf("kubernetes: typed client group %s is not supported", kubernetesResourceLogName(k.Group, k.Version, k.Name))
	}
	groupValues := groupMethod.Call([]reflect.Value{})
	if len(groupValues) == 0 || !groupValues[0].IsValid() {
		return fmt.Errorf("kubernetes: typed client group %s is not supported", kubernetesResourceLogName(k.Group, k.Version, k.Name))
	}
	group := groupValues[0]

	param := []reflect.Value{}
	namespace := ""
	if k.Namespaced {
		param = append(param, reflect.ValueOf(namespace))
	}

	resourceMethod := group.MethodByName(extractClientSetFuncTypeName(k.Name))
	if !resourceMethod.IsValid() {
		return fmt.Errorf("kubernetes: typed client resource %s is not supported", kubernetesResourceLogName(k.Group, k.Version, k.Name))
	}
	resourceValues := resourceMethod.Call(param)
	if len(resourceValues) == 0 || !resourceValues[0].IsValid() {
		return fmt.Errorf("kubernetes: typed client resource %s is not supported", kubernetesResourceLogName(k.Group, k.Version, k.Name))
	}
	resource := resourceValues[0]

	results := resource.MethodByName("List").Call([]reflect.Value{reflect.ValueOf(context.Background()),
		reflect.ValueOf(metav1.ListOptions{})})

	if !results[1].IsNil() {
		return results[1].Interface().(error)
	}
	items := reflect.Indirect(results[0]).FieldByName("Items")
	terraformType := k.terraformType()

	for i := 0; i < items.Len(); i++ {
		item := items.Index(i)
		// Filter to resources that aren't owned by any other resource
		if item.FieldByName("OwnerReferences").Len() > 0 {
			continue
		}

		name := ""
		if k.Namespaced {
			name = item.FieldByName("Namespace").String() + "/" + item.FieldByName("Name").String()
		} else {
			name = item.FieldByName("Name").String()
		}

		k.Resources = append(k.Resources, terraformutils.NewSimpleResource(
			name,
			name,
			terraformType,
			"kubernetes",
			[]string{},
		))
	}
	return nil
}

func (k *Kind) initDynamicResources(client dynamic.Interface) error {
	if k.ResourceName == "" {
		return fmt.Errorf("kubernetes: resource name is required for dynamic resource %s", k.Name)
	}

	resource := client.Resource(schema.GroupVersionResource{
		Group:    k.Group,
		Version:  k.Version,
		Resource: k.ResourceName,
	})

	listClient := dynamic.ResourceInterface(resource)
	if k.Namespaced {
		listClient = resource.Namespace(metav1.NamespaceAll)
	}

	results, err := listClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	terraformType := k.terraformType()
	for i := range results.Items {
		item := results.Items[i]
		// Filter to resources that aren't owned by any other resource.
		if len(item.GetOwnerReferences()) > 0 {
			continue
		}

		name := k.resourceName(item)
		importID := k.importID(item)
		k.Resources = append(k.Resources, terraformutils.NewSimpleResource(
			importID,
			name,
			terraformType,
			"kubernetes",
			[]string{},
		))
	}
	return nil
}

func (k *Kind) resourceName(item unstructured.Unstructured) string {
	name := item.GetName()
	if k.terraformType() != manifestTerraformResourceName {
		if k.Namespaced {
			return item.GetNamespace() + "/" + name
		}
		return name
	}

	parts := []string{k.apiVersion(), k.Name}
	if k.Namespaced {
		parts = append(parts, item.GetNamespace())
	}
	parts = append(parts, name)
	return strings.Join(parts, "/")
}

func (k *Kind) terraformType() string {
	if k.TerraformType != "" {
		return k.TerraformType
	}
	return extractTfResourceName(k.Name)
}

func (k *Kind) importID(item unstructured.Unstructured) string {
	name := item.GetName()
	if k.terraformType() != manifestTerraformResourceName {
		if k.Namespaced {
			return item.GetNamespace() + "/" + name
		}
		return name
	}

	parts := []string{
		"apiVersion=" + k.apiVersion(),
		"kind=" + k.Name,
	}
	if k.Namespaced {
		parts = append(parts, "namespace="+item.GetNamespace())
	}
	parts = append(parts, "name="+name)
	return strings.Join(parts, ",")
}

func (k *Kind) apiVersion() string {
	if k.Group == "" {
		return k.Version
	}
	return k.Group + "/" + k.Version
}
