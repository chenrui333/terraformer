// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	mackerel_terraforming "github.com/chenrui333/terraformer/providers/mackerel"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdMackerelImporter(options ImportOptions) *cobra.Command {
	var apiKey string
	cmd := &cobra.Command{
		Use:   "mackerel",
		Short: "Import current state to Terraform configuration from Mackerel",
		Long:  "Import current state to Terraform configuration from Mackerel",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newMackerelProvider()
			err := Import(provider, options, []string{apiKey})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newMackerelProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "service,role,aws_integration", "aws_integration=id1:id2:id4")
	cmd.PersistentFlags().StringVarP(&apiKey, "api-key", "", "", "YOUR_MACKEREL_API_KEY or env param MACKEREL_API_KEY")
	return cmd
}

func newMackerelProvider() terraformutils.ProviderGenerator {
	return &mackerel_terraforming.MackerelProvider{}
}
