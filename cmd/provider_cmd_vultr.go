// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	vultr_terraforming "github.com/chenrui333/terraformer/providers/vultr"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdVultrImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vultr",
		Short: "Import current state to Terraform configuration from Vultr",
		Long:  "Import current state to Terraform configuration from Vultr",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newVultrProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newVultrProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "instance", "instance=name1:name2:name3")
	return cmd
}

func newVultrProvider() terraformutils.ProviderGenerator {
	return &vultr_terraforming.VultrProvider{}
}
