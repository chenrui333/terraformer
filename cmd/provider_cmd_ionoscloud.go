// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	ionoscloud_terraformer "github.com/chenrui333/terraformer/providers/ionoscloud"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdIonosCloudImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ionoscloud",
		Short: "Import current state to Terraform configuration from IONOS Cloud",
		Long:  "Import current state to Terraform configuration from IONOS Cloud",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newIonosCloudProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newIonosCloudProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "app,addon", "app=name1:name2:name3")
	return cmd
}

func newIonosCloudProvider() terraformutils.ProviderGenerator {
	return &ionoscloud_terraformer.IonosCloudProvider{}
}
