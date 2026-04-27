// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"
	"strings"

	gcp_terraforming "github.com/chenrui333/terraformer/providers/gcp"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdGoogleImporter(options ImportOptions) *cobra.Command {
	providerType := ""
	cmd := &cobra.Command{
		Use:   "google",
		Short: "Import current state to Terraform configuration from Google Cloud",
		Long:  "Import current state to Terraform configuration from Google Cloud",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalPathPattern := options.PathPattern
			for _, project := range options.Projects {
				for _, region := range options.Regions {
					provider := newGoogleProvider()
					options.PathPattern = originalPathPattern
					options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}/{service}", "{provider}/"+project+"/{service}/"+region)
					log.Println(provider.GetName() + " importing project " + project + " region " + region)
					err := Import(provider, options, []string{region, project, providerType})
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newGoogleProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "firewalls,networks", "compute_firewall=id1:id2:id4")
	cmd.PersistentFlags().StringSliceVarP(&options.Regions, "regions", "z", []string{"global"}, "europe-west1,")
	cmd.PersistentFlags().StringSliceVarP(&options.Projects, "projects", "", []string{}, "")
	cmd.PersistentFlags().StringVarP(&providerType, "provider-type", "", "", "beta")
	_ = cmd.MarkPersistentFlagRequired("projects")
	return cmd
}

func newGoogleProvider() terraformutils.ProviderGenerator {
	return &gcp_terraforming.GCPProvider{}
}
