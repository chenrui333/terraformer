// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/chenrui333/terraformer/terraformutils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	secretDataServiceName       = "secretdata"
	secretDataTerraformType     = "kubernetes_secret_v1_data"
	secretDataAllowEmptyPattern = `^data\.`
	secretDataListTimeout       = 30 * time.Second
)

type SecretData struct {
	KubernetesService
	TerraformType string
}

func (s *SecretData) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return s.initResources(clientset)
}

func (s *SecretData) initResources(clientset kubernetes.Interface) error {
	ctx, cancel := context.WithTimeout(context.Background(), secretDataListTimeout)
	defer cancel()

	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	terraformType := s.TerraformType
	if terraformType == "" {
		terraformType = secretDataTerraformType
	}

	for i := range secrets.Items {
		secret := secrets.Items[i]
		if len(secret.Data) == 0 || len(secret.OwnerReferences) > 0 {
			continue
		}

		attributes, ok := secretDataAttributes(secret)
		if !ok {
			continue
		}

		name := secret.Namespace + "/" + secret.Name
		s.Resources = append(s.Resources, terraformutils.NewResource(
			name,
			name,
			terraformType,
			"kubernetes",
			attributes,
			[]string{secretDataAllowEmptyPattern},
			map[string]interface{}{},
		))
	}
	return nil
}

func secretDataAttributes(secret corev1.Secret) (map[string]string, bool) {
	data := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		// kubernetes_secret_v1_data accepts string data only.
		if !utf8.Valid(value) {
			return nil, false
		}
		data[key] = string(value)
	}

	attributes := map[string]string{
		"id":                   secret.Namespace + "/" + secret.Name,
		"metadata.#":           "1",
		"metadata.0.name":      secret.Name,
		"metadata.0.namespace": secret.Namespace,
		"data.%":               strconv.Itoa(len(data)),
	}
	for key, value := range data {
		attributes["data."+key] = value
	}
	return attributes, true
}
