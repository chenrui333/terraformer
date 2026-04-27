// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	ibm_terraforming "github.com/chenrui333/terraformer/providers/ibm"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdIbmImporter(options ImportOptions) *cobra.Command {
	var resourceGroup string
	var region string
	var cis string
	var vpc string
	cmd := &cobra.Command{
		Use:   "ibm",
		Short: "Import current state to Terraform configuration from ibm",
		Long:  "Import current state to Terraform configuration from ibm",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newIbmProvider()
			err := Import(provider, options, []string{resourceGroup, region, cis, vpc})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newIbmProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "server", "ibm_server=name1:name2:name3")
	cmd.PersistentFlags().StringVarP(&resourceGroup, "resource_group", "", "", "resource_group=default")
	cmd.PersistentFlags().StringVarP(&region, "region", "R", "", "region=us-south")
	cmd.PersistentFlags().StringVarP(&cis, "cis", "", "", "cis=TestCIS")
	cmd.PersistentFlags().StringVarP(&vpc, "vpc", "", "", "vpc=vpc01")
	return cmd
}

func newIbmProvider() terraformutils.ProviderGenerator {
	return &ibm_terraforming.IBMProvider{}
}
