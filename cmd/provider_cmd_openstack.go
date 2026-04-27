// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"

	openstack_terraforming "github.com/chenrui333/terraformer/providers/openstack"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdOpenStackImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openstack",
		Short: "Import current state to Terraform configuration from OpenStack",
		Long:  "Import current state to Terraform configuration from OpenStack",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, region := range options.Regions {
				provider := newOpenStackProvider()
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
	cmd.AddCommand(listCmd(newOpenStackProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "compute,networking", "compute_instance_v2=id1:id2:id4")
	cmd.PersistentFlags().StringSliceVarP(&options.Regions, "regions", "", []string{}, "RegionOne")
	return cmd
}

func newOpenStackProvider() terraformutils.ProviderGenerator {
	return &openstack_terraforming.OpenStackProvider{}
}
