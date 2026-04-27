// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	newrelic_terraforming "github.com/chenrui333/terraformer/providers/newrelic"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdNewRelicImporter(options ImportOptions) *cobra.Command {
	apiKey := ""
	accountID := ""
	region := ""
	cmd := &cobra.Command{
		Use:   "newrelic",
		Short: "Import current state to Terraform configuration from New Relic",
		Long:  "Import current state to Terraform configuration from New Relic",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newNewRelicProvider()
			err := Import(provider, options, []string{apiKey, accountID, region})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newNewRelicProvider()))
	cmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Your Personal API Key")
	cmd.PersistentFlags().StringVar(&accountID, "account-id", "", "Your Account ID")
	cmd.PersistentFlags().StringVar(&region, "region", "US", "")
	baseProviderFlags(cmd.PersistentFlags(), &options, "alert", "dashboard=id1:id2:id4")
	return cmd
}

func newNewRelicProvider() terraformutils.ProviderGenerator {
	return &newrelic_terraforming.NewRelicProvider{}
}
