// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"log"

	awsterraformer "github.com/chenrui333/terraformer/providers/aws"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdAwsImporter(options ImportOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Import current state to Terraform configuration from AWS",
		Long:  "Import current state to Terraform configuration from AWS",
		RunE: func(_ *cobra.Command, _ []string) error {
			originalResources := options.Resources
			originalRegions := options.Regions
			originalPathPattern := options.PathPattern

			if len(options.Regions) > 0 {
				shouldSpecifyPathRegion := len(options.Regions) > 1
				globalResources, eastOnlyResources, chatbotResources, regionalResources := parseAndGroupResources(originalResources)
				options.Resources = globalResources
				options.Regions = []string{awsterraformer.GlobalRegion}
				e := importGlobalResources(options)
				if e != nil {
					return e
				}

				options.Resources = eastOnlyResources
				options.Regions = []string{awsterraformer.MainRegionPublicPartition}
				e = importEastOnlyResources(options)
				if e != nil {
					return e
				}

				chatbotShouldSpecifyPathRegion := shouldSpecifyPathRegion || len(globalResources) > 0 || len(eastOnlyResources) > 0 || len(regionalResources) > 0
				options.Resources = chatbotResources
				options.Regions = chatbotImportRegions(originalRegions)
				e = importChatbotResources(options, originalPathPattern, chatbotShouldSpecifyPathRegion)
				if e != nil {
					return e
				}

				options.Resources = regionalResources
				options.Regions = originalRegions
				if len(options.Resources) > 0 { // don't import anything and potentially override global resources
					if len(globalResources) > 0 {
						shouldSpecifyPathRegion = true // we should keep global resources away from regional
					}
					for _, region := range originalRegions {
						e := importRegionResources(options, originalPathPattern, region, shouldSpecifyPathRegion)
						if e != nil {
							return e
						}
					}
				}
				return nil
			}
			err := importRegionResources(options, options.PathPattern, awsterraformer.NoRegion, false)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(listCmd(newAWSProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "vpc,subnet,nacl", "elb=id1:id2:id4")

	cmd.PersistentFlags().StringVarP(&options.Profile, "profile", "", "default", "prod")
	cmd.PersistentFlags().StringSliceVarP(&options.Regions, "regions", "", []string{}, "eu-west-1,eu-west-2,us-east-1")
	return cmd
}

// returns global, east-only, chatbot, regional resources
func parseAndGroupResources(allResources []string) ([]string, []string, []string, []string) {
	var globalResources, eastOnlyResources, chatbotResources, regionalResources []string
	for _, resourceName := range allResources {
		switch {
		case contains(awsterraformer.SupportedGlobalResources, resourceName):
			globalResources = append(globalResources, resourceName)
		case contains(awsterraformer.SupportedEastOnlyResources, resourceName):
			eastOnlyResources = append(eastOnlyResources, resourceName)
		case contains(awsterraformer.SupportedChatbotResources, resourceName):
			chatbotResources = append(chatbotResources, resourceName)
		default:
			regionalResources = append(regionalResources, resourceName)
		}
	}
	return globalResources, eastOnlyResources, chatbotResources, regionalResources
}

func importGlobalResources(options ImportOptions) error {
	if len(options.Resources) > 0 {
		return importRegionResources(options, options.PathPattern, awsterraformer.GlobalRegion, false)
	}
	return nil
}

func importEastOnlyResources(options ImportOptions) error {
	if len(options.Resources) > 0 {
		return importRegionResources(options, options.PathPattern, awsterraformer.MainRegionPublicPartition, false)
	}
	return nil
}

func importChatbotResources(options ImportOptions, originalPathPattern string, shouldSpecifyPathRegion bool) error {
	if len(options.Resources) == 0 {
		return nil
	}
	for _, region := range options.Regions {
		if err := importRegionResources(options, originalPathPattern, region, shouldSpecifyPathRegion); err != nil {
			return err
		}
	}
	return nil
}

func chatbotImportRegions(regions []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(regions))
	for _, region := range regions {
		effectiveRegion := awsterraformer.ChatbotAPIRegion(region)
		if seen[effectiveRegion] {
			continue
		}
		seen[effectiveRegion] = true
		result = append(result, effectiveRegion)
	}
	return result
}

func importRegionResources(options ImportOptions, originalPathPattern string, region string, shouldSpecifyPathRegion bool) error {
	provider := newAWSProvider()
	options.PathPattern = originalPathPattern
	if region != awsterraformer.GlobalRegion && region != awsterraformer.NoRegion {
		if shouldSpecifyPathRegion {
			options.PathPattern += region + "/"
		}
		log.Println(provider.GetName() + " importing region " + region)
	} else {
		log.Println(provider.GetName() + " importing default region")
	}
	err := Import(provider, options, []string{region, options.Profile})
	if err != nil {
		return err
	}
	return nil
}

func newAWSProvider() terraformutils.ProviderGenerator {
	return &awsterraformer.AWSProvider{}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
