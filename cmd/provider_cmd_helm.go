// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	helm_terraforming "github.com/chenrui333/terraformer/providers/helm"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdHelmImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm",
		Short: "Import current state to Terraform configuration from Helm",
		Long:  "Import current state to Terraform configuration from Helm",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newHelmProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newHelmProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "release", "release=namespace/name")
	return cmd
}

func newHelmProvider() terraformutils.ProviderGenerator {
	return &helm_terraforming.Provider{}
}
