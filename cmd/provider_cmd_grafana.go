// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"github.com/chenrui333/terraformer/providers/grafana"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdGrafanaImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grafana",
		Short: "Import current state to Terraform configuration from Grafana",
		Long:  "Import current state to Terraform configuration from Grafana",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newGrafanaProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newGrafanaProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "grafana_dashboard", "dashboard=slug1")
	return cmd
}

func newGrafanaProvider() terraformutils.ProviderGenerator {
	return &grafana.GrafanaProvider{}
}
