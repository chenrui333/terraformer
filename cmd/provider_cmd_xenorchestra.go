// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	xenorchestra_terraforming "github.com/chenrui333/terraformer/providers/xenorchestra"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdXenorchestraImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "xenorchestra",
		Short: "Import current state to Terraform configuration from Xen Orchestra",
		Long:  "Import current state to Terraform configuration from Xen Orchestra",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newXenorchestraProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newXenorchestraProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "instance", "acl=name1:name2:name3")
	return cmd
}

func newXenorchestraProvider() terraformutils.ProviderGenerator {
	return &xenorchestra_terraforming.XenorchestraProvider{}
}
