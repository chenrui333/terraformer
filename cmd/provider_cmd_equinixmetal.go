// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	equinixmetal_terraforming "github.com/chenrui333/terraformer/providers/equinixmetal"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdEquinixMetalImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metal",
		Short: "Import current state to Terraform configuration from Equinix Metal",
		Long:  "Import current state to Terraform configuration from Equinix Metal",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newEquinixMetalProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newEquinixMetalProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "project,device", "project=name1:name2:name3")

	return cmd
}

func newEquinixMetalProvider() terraformutils.ProviderGenerator {
	return &equinixmetal_terraforming.EquinixMetalProvider{}
}
