// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	configMapDataServiceName       = "configmapdata"
	configMapDataTerraformType     = "kubernetes_config_map_v1_data"
	configMapDataAllowEmptyPattern = `^data\.`
)

type ConfigMapData struct {
	KubernetesService
	TerraformType string
}

func (c *ConfigMapData) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return c.initResources(clientset)
}

func (c *ConfigMapData) initResources(clientset kubernetes.Interface) error {
	configMaps, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	terraformType := c.TerraformType
	if terraformType == "" {
		terraformType = configMapDataTerraformType
	}

	for i := range configMaps.Items {
		configMap := configMaps.Items[i]
		if len(configMap.Data) == 0 || len(configMap.OwnerReferences) > 0 {
			continue
		}

		name := configMap.Namespace + "/" + configMap.Name
		c.Resources = append(c.Resources, terraformutils.NewResource(
			name,
			name,
			terraformType,
			"kubernetes",
			configMapDataAttributes(configMap),
			[]string{configMapDataAllowEmptyPattern},
			map[string]interface{}{},
		))
	}
	return nil
}

func configMapDataAttributes(configMap corev1.ConfigMap) map[string]string {
	attributes := map[string]string{
		"id":                   configMap.Namespace + "/" + configMap.Name,
		"metadata.#":           "1",
		"metadata.0.name":      configMap.Name,
		"metadata.0.namespace": configMap.Namespace,
		"data.%":               strconv.Itoa(len(configMap.Data)),
	}
	for key, value := range configMap.Data {
		attributes["data."+key] = value
	}
	return attributes
}
