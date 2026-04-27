// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	honeycombio_terraforming "github.com/chenrui333/terraformer/providers/honeycombio"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdHoneycombioImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "honeycombio",
		Short: "Import current state to Terraform configuration from Honeycomb.io",
		Long:  "Import current state to Terraform configuration from Honeycomb.io",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newHoneycombioProvider()
			err := Import(provider, options, options.Projects)
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newHoneycombioProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "derived_column,board", "board=id1,id2")
	cmd.PersistentFlags().StringSliceVarP(&options.Projects, "datasets", "", []string{}, "hello-service,goodbye-service")

	return cmd
}

func newHoneycombioProvider() terraformutils.ProviderGenerator {
	return &honeycombio_terraforming.HoneycombProvider{}
}
