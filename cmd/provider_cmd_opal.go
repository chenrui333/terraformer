// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	opal_terraformer "github.com/chenrui333/terraformer/providers/opal"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdOpalImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opal",
		Short: "Import current state to Terraform configuration from opal.dev",
		Long:  "Import current state to Terraform configuration from opal.dev",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newOpalProvider()
			err := Import(provider, options, options.Projects)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newOpalProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "", "")

	return cmd
}

func newOpalProvider() terraformutils.ProviderGenerator {
	return &opal_terraformer.OpalProvider{}
}
