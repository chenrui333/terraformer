// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	launchdarkly_terraforming "github.com/chenrui333/terraformer/providers/launchdarkly"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdLaunchDarklyImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launchdarkly",
		Short: "Import current state to Terraform configuration from LaunchDarkly",
		Long:  "Import current state to Terraform configuration from LaunchDarkly",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newLaunchDarklyProvider()
			err := Import(provider, options, []string{})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newLaunchDarklyProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "project", "launchdarkly_project=id1:id2:id3")
	return cmd
}

func newLaunchDarklyProvider() terraformutils.ProviderGenerator {
	return &launchdarkly_terraforming.LaunchDarklyProvider{}
}
