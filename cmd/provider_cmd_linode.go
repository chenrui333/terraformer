// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	linode_terraforming "github.com/chenrui333/terraformer/providers/linode"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdLinodeImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "linode",
		Short: "Import current state to Terraform configuration from Linode",
		Long:  "Import current state to Terraform configuration from Linode",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newLinodeProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newLinodeProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "instance", "instance=name1:name2:name3")
	return cmd
}

func newLinodeProvider() terraformutils.ProviderGenerator {
	return &linode_terraforming.LinodeProvider{}
}
