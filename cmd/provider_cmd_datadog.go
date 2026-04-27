// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	datadog_terraforming "github.com/chenrui333/terraformer/providers/datadog"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdDatadogImporter(options ImportOptions) *cobra.Command {
	var apiKey, appKey, apiURL, validate string
	cmd := &cobra.Command{
		Use:   "datadog",
		Short: "Import current state to Terraform configuration from Datadog",
		Long:  "Import current state to Terraform configuration from Datadog",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newDataDogProvider()
			err := Import(provider, options, []string{apiKey, appKey, apiURL, validate})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newDataDogProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "monitors,users", "monitor=id1:id2:id4")
	cmd.PersistentFlags().StringVarP(&apiKey, "api-key", "", "", "YOUR_DATADOG_API_KEY or env param DATADOG_API_KEY")
	cmd.PersistentFlags().StringVarP(&appKey, "app-key", "", "", "YOUR_DATADOG_APP_KEY or env param DATADOG_APP_KEY")
	cmd.PersistentFlags().StringVarP(&apiURL, "api-url", "", "", "YOUR_DATADOG_API_URL or env param DATADOG_HOST")
	cmd.PersistentFlags().StringVar(&validate, "validate", "", "bool-parsable values only or env param DATADOG_VALIDATE. Enables validation of the provided API and APP keys during provider initialization. Default is true. When false, api_key and app_key won't be checked")
	return cmd
}

func newDataDogProvider() terraformutils.ProviderGenerator {
	return &datadog_terraforming.DatadogProvider{}
}
