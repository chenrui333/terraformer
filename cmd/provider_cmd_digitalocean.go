// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	digitalocean_terraforming "github.com/chenrui333/terraformer/providers/digitalocean"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdDigitalOceanImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digitalocean",
		Short: "Import current state to Terraform configuration from DigitalOcean",
		Long:  "Import current state to Terraform configuration from DigitalOcean",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newDigitalOceanProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newDigitalOceanProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "project,droplet", "project=name1:name2:name3")

	return cmd
}

func newDigitalOceanProvider() terraformutils.ProviderGenerator {
	return &digitalocean_terraforming.DigitalOceanProvider{}
}
