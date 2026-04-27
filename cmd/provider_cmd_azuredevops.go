// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	azuredevops "github.com/chenrui333/terraformer/providers/azuredevops"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAzureDevOpsImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "azuredevops",
		Short: "Import current state to Terraform configuration from Azure DevOps",
		Long:  "Import current state to Terraform configuration from Azure DevOps",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newAzureDevOpsProvider()
			err := Import(provider, options, []string{options.ResourceGroup})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newAzureDevOpsProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "project,team,git", "project=name1:name2:name3")
	return cmd
}

func newAzureDevOpsProvider() terraformutils.ProviderGenerator {
	return &azuredevops.AzureDevOpsProvider{}
}
