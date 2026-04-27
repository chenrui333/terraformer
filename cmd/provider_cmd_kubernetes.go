// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"strconv"

	kubernetes_terraforming "github.com/chenrui333/terraformer/providers/kubernetes"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdKubernetesImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubernetes",
		Short: "Import current state to Terraform configuration from Kubernetes",
		Long:  "Import current state to Terraform configuration from Kubernetes",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newKubernetesProvider()
			err := Import(provider, options, []string{strconv.FormatBool(options.Verbose)})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newKubernetesProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "configmaps,deployments,services", "deployment=name1:name2:name3")
	return cmd
}

func newKubernetesProvider() terraformutils.ProviderGenerator {
	return &kubernetes_terraforming.KubernetesProvider{}
}
