// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	cloudflare_terraforming "github.com/chenrui333/terraformer/providers/cloudflare"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdCloudflareImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloudflare",
		Short: "Import current state to Terraform configuration from Cloudflare",
		Long:  "Import current state to Terraform configuration from Cloudflare",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newCloudflareProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newCloudflareProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "zone", "access_application=id1:id2:id4")
	return cmd
}

func newCloudflareProvider() terraformutils.ProviderGenerator {
	return &cloudflare_terraforming.CloudflareProvider{}
}
