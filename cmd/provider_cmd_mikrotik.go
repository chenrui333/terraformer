// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	mikrotik_terraforming "github.com/chenrui333/terraformer/providers/mikrotik"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdMikrotikImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mikrotik",
		Short: "Import current state to Terraform configuration from RouterOS",
		Long:  "Import current state to Terraform configuration from RouterOS",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newMikrotikProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newMikrotikProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "instance", "dhcp_lease=name1:name2:name3")
	return cmd
}

func newMikrotikProvider() terraformutils.ProviderGenerator {
	return &mikrotik_terraforming.MikrotikProvider{}
}
