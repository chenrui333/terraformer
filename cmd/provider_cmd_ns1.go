// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	ns1_terraforming "github.com/chenrui333/terraformer/providers/ns1"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdNs1Importer(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ns1",
		Short: "Import current state to Terraform configuration from NS1",
		Long:  "Import current state to Terraform configuration from NS1",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newNs1Provider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newNs1Provider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "zone", "zone=id1:id2:id4")
	return cmd
}

func newNs1Provider() terraformutils.ProviderGenerator {
	return &ns1_terraforming.Ns1Provider{}
}
