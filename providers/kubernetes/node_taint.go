// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	nodeTaintServiceName       = "nodetaints"
	nodeTaintTerraformType     = "kubernetes_node_taint"
	nodeTaintAllowEmptyPattern = `^taint\.[0-9]+\.value$`
)

type NodeTaint struct {
	KubernetesService
	TerraformType string
}

func (n *NodeTaint) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return n.initResources(clientset)
}

func (n *NodeTaint) initResources(clientset kubernetes.Interface) error {
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	terraformType := n.TerraformType
	if terraformType == "" {
		terraformType = nodeTaintTerraformType
	}

	for i := range nodes.Items {
		node := nodes.Items[i]
		if len(node.Spec.Taints) == 0 {
			continue
		}

		attributes := nodeTaintAttributes(node.Name, node.Spec.Taints)
		n.Resources = append(n.Resources, terraformutils.NewResource(
			attributes["id"],
			node.Name,
			terraformType,
			"kubernetes",
			attributes,
			[]string{nodeTaintAllowEmptyPattern},
			map[string]interface{}{},
		))
	}
	return nil
}

func nodeTaintAttributes(nodeName string, taints []corev1.Taint) map[string]string {
	attributes := map[string]string{
		"id":              nodeTaintID(nodeName, taints),
		"metadata.#":      "1",
		"metadata.0.name": nodeName,
		"taint.#":         strconv.Itoa(len(taints)),
	}
	for i, taint := range taints {
		prefix := "taint." + strconv.Itoa(i) + "."
		attributes[prefix+"effect"] = string(taint.Effect)
		attributes[prefix+"key"] = taint.Key
		attributes[prefix+"value"] = taint.Value
	}
	return attributes
}

func nodeTaintID(nodeName string, taints []corev1.Taint) string {
	id := nodeName
	for _, taint := range taints {
		id += fmt.Sprintf(",%s=%s:%s", taint.Key, taint.Value, taint.Effect)
	}
	return id
}
