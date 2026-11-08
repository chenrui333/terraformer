// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	pagerduty_terraforming "github.com/chenrui333/terraformer/providers/pagerduty"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdPagerDutyImporter(options ImportOptions) *cobra.Command {
	token := ""
	cmd := &cobra.Command{
		Use:   "pagerduty",
		Short: "Import current state to Terraform configuration from PagerDuty",
		Long:  "Import current state to Terraform configuration from PagerDuty",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newPagerDutyProvider()
			err := Import(provider, options, []string{token})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newPagerDutyProvider()))
	cmd.PersistentFlags().StringVarP(&token, "token", "t", "", "PagerDuty API token for discovery/refresh only (or env PAGERDUTY_TOKEN); generated provider.tf omits it, so set PAGERDUTY_TOKEN for terraform")
	baseProviderFlags(cmd.PersistentFlags(), &options, "user", "user=id1:id2:id4")
	return cmd
}

func newPagerDutyProvider() terraformutils.ProviderGenerator {
	return &pagerduty_terraforming.PagerDutyProvider{}
}
