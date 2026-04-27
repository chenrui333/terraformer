// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log"

	tencentcloud_terraforming "github.com/chenrui333/terraformer/providers/tencentcloud"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdTencentCloudImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tencentcloud",
		Short: "Import current state to Terraform configuration from Tencent Cloud",
		Long:  "Import current state to Terraform configuration from Tencent Cloud",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, region := range options.Regions {
				provider := newTencentCloudProvider()
				options.PathPattern = originalPathPattern
				options.PathPattern += region + "/"
				log.Println(provider.GetName() + " importing region " + region)
				err := Import(provider, options, []string{region})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newTencentCloudProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "cvm,vpc,cdn", "tencentcloud_vpc=id1:id2:id3")
	cmd.PersistentFlags().StringSliceVarP(&options.Regions, "regions", "", []string{}, "ap-guangzhou")
	return cmd
}

func newTencentCloudProvider() terraformutils.ProviderGenerator {
	return &tencentcloud_terraforming.TencentCloudProvider{}
}
