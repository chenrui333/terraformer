// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultServiceAccountKind        = "DefaultServiceAccount"
	defaultServiceAccountName        = "default"
	defaultServiceAccountServiceName = "defaultserviceaccounts"
)

type DefaultServiceAccount struct {
	KubernetesService
	TerraformType string
}

func (d *DefaultServiceAccount) InitResources() error {
	config, _, err := initClientAndConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return d.initResources(clientset)
}

func (d *DefaultServiceAccount) initResources(clientset kubernetes.Interface) error {
	accounts, err := clientset.CoreV1().ServiceAccounts(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	terraformType := d.TerraformType
	if terraformType == "" {
		terraformType = extractTfResourceName(defaultServiceAccountKind)
	}

	for i := range accounts.Items {
		account := accounts.Items[i]
		if account.Name != defaultServiceAccountName || len(account.OwnerReferences) > 0 {
			continue
		}

		name := account.Namespace + "/" + account.Name
		d.Resources = append(d.Resources, terraformutils.NewSimpleResource(
			name,
			name,
			terraformType,
			"kubernetes",
			[]string{},
		))
	}
	return nil
}
