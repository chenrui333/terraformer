// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strconv"

	kafka_terraforming "github.com/chenrui333/terraformer/providers/kafka"
	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/spf13/cobra"
)

func newCmdKafkaImporter(options ImportOptions) *cobra.Command {
	config := kafka_terraforming.ConfigFromEnv()
	cmd := &cobra.Command{
		Use:   "kafka",
		Short: "Import current state to Terraform configuration from Kafka",
		Long:  "Import current state to Terraform configuration from Kafka",
		RunE: func(_ *cobra.Command, _ []string) error {
			provider := newKafkaProvider()
			err := Import(provider, options, []string{kafka_terraforming.EncodeConfig(config)})
			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.AddCommand(listCmd(newKafkaProvider()))
	baseProviderFlags(cmd.PersistentFlags(), &options, "topics", "topic=topic1:topic2")
	cmd.PersistentFlags().StringSliceVar(&config.BootstrapServers, "bootstrap-servers", config.BootstrapServers, "Kafka bootstrap servers or KAFKA_BOOTSTRAP_SERVERS")
	cmd.PersistentFlags().StringVar(&config.KafkaVersion, "kafka-version", config.KafkaVersion, "Kafka protocol version or KAFKA_VERSION")
	cmd.PersistentFlags().BoolVar(&config.TLSEnabled, "tls-enabled", config.TLSEnabled, "Enable TLS or KAFKA_ENABLE_TLS")
	cmd.PersistentFlags().BoolVar(&config.SkipTLSVerify, "skip-tls-verify", config.SkipTLSVerify, "Skip TLS certificate verification or KAFKA_SKIP_VERIFY")
	cmd.PersistentFlags().StringVar(&config.SASLMechanism, "sasl-mechanism", config.SASLMechanism, "SASL mechanism or KAFKA_SASL_MECHANISM")
	cmd.PersistentFlags().StringVar(&config.SASLUsername, "sasl-username", config.SASLUsername, "SASL username or KAFKA_SASL_USERNAME")
	cmd.PersistentFlags().StringVar(&config.SASLAWSRegion, "sasl-aws-region", config.SASLAWSRegion, "AWS region for aws-iam SASL or KAFKA_SASL_IAM_AWS_REGION")
	cmd.PersistentFlags().StringVar(&config.SASLAWSProfile, "sasl-aws-profile", config.SASLAWSProfile, "AWS profile for aws-iam SASL or AWS_PROFILE")
	cmd.PersistentFlags().StringVar(&config.SASLAWSRoleARN, "sasl-aws-role-arn", config.SASLAWSRoleARN, "AWS role ARN for aws-iam SASL or AWS_ROLE_ARN")
	cmd.PersistentFlags().StringVar(&config.SASLAWSExternalID, "sasl-aws-external-id", config.SASLAWSExternalID, "AWS external ID for aws-iam SASL")
	cmd.PersistentFlags().StringSliceVar(&config.SASLAWSSharedConfigFiles, "sasl-aws-shared-config-files", config.SASLAWSSharedConfigFiles, "AWS shared config files for aws-iam SASL")
	cmd.PersistentFlags().StringVar(&config.SASLTokenURL, "sasl-token-url", config.SASLTokenURL, "OAuth token URL or KAFKA_SASL_TOKEN_URL")
	cmd.PersistentFlags().StringSliceVar(&config.SASLOAuthScopes, "sasl-oauth-scopes", config.SASLOAuthScopes, "OAuth scopes or KAFKA_SASL_OAUTH_SCOPES")
	cmd.PersistentFlags().StringVar(&config.CACert, "ca-cert", config.CACert, "CA certificate PEM or file path, or KAFKA_CA_CERT")
	cmd.PersistentFlags().StringVar(&config.ClientCert, "client-cert", config.ClientCert, "Client certificate PEM or file path, or KAFKA_CLIENT_CERT")
	cmd.PersistentFlags().IntVar(&config.Timeout, "timeout", config.Timeout, "Kafka admin timeout in seconds or KAFKA_TIMEOUT")
	cmd.PersistentFlags().Lookup("timeout").DefValue = strconv.Itoa(config.Timeout)
	return cmd
}

func newKafkaProvider() terraformutils.ProviderGenerator {
	return &kafka_terraforming.Provider{}
}
