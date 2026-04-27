// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	fastly_terraforming "github.com/chenrui333/terraformer/providers/fastly"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdFastlyImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fastly",
		Short: "Import current state to Terraform configuration from Fastly",
		Long:  "Import current state to Terraform configuration from Fastly",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newFastlyProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newFastlyProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "service_v1", "service_v1=id1:id2:id3")
	return cmd
}

func newFastlyProvider() terraformutils.ProviderGenerator {
	return &fastly_terraforming.FastlyProvider{}
}
