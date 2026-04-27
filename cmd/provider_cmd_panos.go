// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"log"
	"reflect"
	"strings"

	panos_terraforming "github.com/chenrui333/terraformer/providers/panos"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdPanosImporter(options ImportOptions) *cobra.Command {
	vsys := []string{}
	cmd := &cobra.Command{
		Use:   "panos",
		Short: "Import current state to Terraform configuration from a PAN-OS",
		Long:  "Import current state to Terraform configuration from a PAN-OS",
		RunE: func(_ *cobra.Command, _ []string) error {
			var t interface{}

			if len(vsys) == 0 {
				var err error

				vsys, t, err = panos_terraforming.GetVsysList()
				if err != nil {
					return err
				}
			} else {
				c, err := panos_terraforming.Initialize()
				if err != nil {
					return err
				}

				t = reflect.TypeOf(c)
			}

			resources := panos_terraforming.FilterCallableResources(t, options.Resources)
			options.Resources = resources

			originalPathPattern := options.PathPattern
			for _, v := range vsys {
				provider := newPanosProvider()
				log.Println(provider.GetName() + " importing VSYS " + v)
				options.PathPattern = originalPathPattern
				options.PathPattern = strings.ReplaceAll(options.PathPattern, "{provider}", "{provider}/"+v)

				err := Import(provider, options, []string{v})
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.AddCommand(listCmd(newPanosProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "firewall_device_config,firewall_networking,firewall_objects,firewall_policy", "")
	cmd.PersistentFlags().StringSliceVarP(&vsys, "vsys", "", []string{}, "")

	return cmd
}

func newPanosProvider() terraformutils.ProviderGenerator {
	return &panos_terraforming.PanosProvider{}
}
