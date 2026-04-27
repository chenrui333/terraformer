// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	azure_terraforming "github.com/chenrui333/terraformer/providers/azure"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAzureImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azure",
		Short: "Import current state to Terraform configuration from Azure",
		Long:  "Import current state to Terraform configuration from Azure",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newAzureProvider()
			err := Import(provider, options, []string{options.ResourceGroup})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newAzureProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "resource_group", "resource_group=name1:name2:name3")
	cmd.PersistentFlags().StringVarP(&options.ResourceGroup, "resource-group", "R", "", "")
	return cmd
}

func newAzureProvider() terraformutils.ProviderGenerator {
	return &azure_terraforming.AzureProvider{}
}
