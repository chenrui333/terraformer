// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	azuread "github.com/chenrui333/terraformer/providers/azuread"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAzureADImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azuread",
		Short: "Import current state to Terraform configuration from Azure Active Directory",
		Long:  "Import current state to Terraform configuration from Azure Active Directory",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newAzureADProvider()
			err := Import(provider, options, []string{options.ResourceGroup})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newAzureADProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "resource_group", "resource_group=name1:name2:name3")
	cmd.PersistentFlags().StringVarP(&options.ResourceGroup, "resource-group", "R", "", "")
	return cmd
}

func newAzureADProvider() terraformutils.ProviderGenerator {
	return &azuread.AzureADProvider{}
}
