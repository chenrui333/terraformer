// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/chenrui333/terraformer/terraformutils"
	terraformerVersion "github.com/chenrui333/terraformer/version"
	"github.com/spf13/cobra"
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       terraformerVersion.Version,
	}
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newPlanCmd())
	cmd.AddCommand(versionCmd)
	return cmd
}

func Execute() error {
	cmd := NewCmdRoot()
	return cmd.Execute()
}

func providerImporterSubcommands() []func(options ImportOptions) *cobra.Command {
	return []func(options ImportOptions) *cobra.Command{
		// Major Cloud
		newCmdGoogleImporter,
		newCmdAwsImporter,
		newCmdAzureImporter,
		newCmdAliCloudImporter,
		newCmdIbmImporter,
		// Cloud
		newCmdDigitalOceanImporter,
		newCmdEquinixMetalImporter,
		newCmdHerokuImporter,
		newCmdLaunchDarklyImporter,
		newCmdLinodeImporter,
		newCmdOpenStackImporter,
		newCmdTencentCloudImporter,
		newCmdVultrImporter,
		newCmdYandexImporter,
		newCmdIonosCloudImporter,
		// Infrastructure Software
		newCmdKubernetesImporter,
		newCmdOctopusDeployImporter,
		newCmdRabbitMQImporter,
		// Network
		newCmdMyrasecImporter,
		newCmdCloudflareImporter,
		newCmdFastlyImporter,
		newCmdNs1Importer,
		newCmdPanosImporter,
		// VCS
		newCmdAzureDevOpsImporter,
		newCmdAzureADImporter,
		newCmdGithubImporter,
		newCmdGitLabImporter,
		// Monitoring & System Management
		newCmdDatadogImporter,
		newCmdNewRelicImporter,
		newCmdMackerelImporter,
		newCmdGrafanaImporter,
		newCmdPagerDutyImporter,
		newCmdOpsgenieImporter,
		newCmdHoneycombioImporter,
		newCmdOpalImporter,
		// Community
		newCmdKeycloakImporter,
		newCmdLogzioImporter,
		newCmdCommercetoolsImporter,
		newCmdMikrotikImporter,
		newCmdXenorchestraImporter,
		newCmdGmailfilterImporter,
		newCmdVaultImporter,
		newCmdOktaImporter,
		newCmdAuth0Importer,
	}
}

func providerGenerators() map[string]func() terraformutils.ProviderGenerator {
	list := make(map[string]func() terraformutils.ProviderGenerator)
	for _, providerGen := range []func() terraformutils.ProviderGenerator{
		// Major Cloud
		newGoogleProvider,
		newAWSProvider,
		newAzureProvider,
		newAliCloudProvider,
		newIbmProvider,
		// Cloud
		newDigitalOceanProvider,
		newEquinixMetalProvider,
		newFastlyProvider,
		newHerokuProvider,
		newIonosCloudProvider,
		newLaunchDarklyProvider,
		newLinodeProvider,
		newNs1Provider,
		newOpenStackProvider,
		newTencentCloudProvider,
		newVultrProvider,
		newYandexProvider,
		// Infrastructure Software
		newKubernetesProvider,
		newOctopusDeployProvider,
		newRabbitMQProvider,
		// Network
		newMyrasecProvider,
		newCloudflareProvider,
		newPanosProvider,
		// VCS
		newAzureDevOpsProvider,
		newAzureADProvider,
		newGitHubProvider,
		newGitLabProvider,
		// Monitoring & System Management
		newDataDogProvider,
		newGrafanaProvider,
		newNewRelicProvider,
		newMackerelProvider,
		newOpsgenieProvider,
		newPagerDutyProvider,
		newHoneycombioProvider,
		newOpalProvider,
		// Community
		newKeycloakProvider,
		newLogzioProvider,
		newCommercetoolsProvider,
		newMikrotikProvider,
		newXenorchestraProvider,
		newGmailfilterProvider,
		newVaultProvider,
		newOktaProvider,
		newAuth0Provider,
	} {
		list[providerGen().GetName()] = providerGen
	}
	return list
}
