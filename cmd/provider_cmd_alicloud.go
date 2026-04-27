// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"

	alicloud_terraforming "github.com/chenrui333/terraformer/providers/alicloud"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAliCloudImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alicloud",
		Short: "Import current State to terraform configuration from alicloud",
		Long:  "Import current State to terraform configuration from alicloud",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, region := range options.Regions {
				provider := newAliCloudProvider()
				options.PathPattern = originalPathPattern
				options.PathPattern += region + "/"
				log.Println(provider.GetName() + " importing region " + region)
				profile := options.Profile
				err := Import(provider, options, []string{region, profile})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newAliCloudProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "vpc,subnet,nacl", "slb=id1:id2:id4")
	cmd.PersistentFlags().StringVar(&options.Profile, "profile", "default", "prod")
	cmd.PersistentFlags().StringSliceVarP(&options.Regions, "regions", "", []string{}, "cn-hangzhou")
	return cmd
}

func newAliCloudProvider() terraformutils.ProviderGenerator {
	return &alicloud_terraforming.AliCloudProvider{}
}
