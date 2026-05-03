// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"

	"github.com/chenrui333/terraformer/terraformutils"
)

type AWSService struct { //nolint
	terraformutils.Service
}

var awsVariable = regexp.MustCompile(`(\${[0-9A-Za-z:]+})`)

var configCache *aws.Config

func (s *AWSService) generateConfig() (aws.Config, error) {
	if configCache != nil {
		return *configCache, nil
	}

	baseConfig, e := s.buildBaseConfig()

	if e != nil {
		return baseConfig, e
	}
	if s.Verbose {
		baseConfig.ClientLogMode = aws.LogRequestWithBody & aws.LogResponseWithBody
	}

	creds, e := baseConfig.Credentials.Retrieve(context.TODO())

	if e != nil {
		return baseConfig, e
	}

	// terraform cannot ask for MFA token, so we need to pass STS session token, which might contain credentials with MFA requirement
	accessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKey == "" {
		if err := terraformutils.SetEnv("AWS_ACCESS_KEY_ID", creds.AccessKeyID); err != nil {
			return baseConfig, err
		}
		if err := terraformutils.SetEnv("AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey); err != nil {
			return baseConfig, err
		}

		if creds.SessionToken != "" {
			if err := terraformutils.SetEnv("AWS_SESSION_TOKEN", creds.SessionToken); err != nil {
				return baseConfig, err
			}
		}
	}
	configCache = &baseConfig
	return baseConfig, nil
}

func (s *AWSService) buildBaseConfig() (aws.Config, error) {
	var loadOptions []func(*config.LoadOptions) error
	if s.GetArgs()["profile"].(string) != "" {
		loadOptions = append(loadOptions, config.WithSharedConfigProfile(s.GetArgs()["profile"].(string)))
	}
	if s.GetArgs()["region"].(string) != "" {
		if err := terraformutils.SetEnv("AWS_REGION", s.GetArgs()["region"].(string)); err != nil {
			return aws.Config{}, err
		}
	}
	loadOptions = append(loadOptions, config.WithAssumeRoleCredentialOptions(func(options *stscreds.AssumeRoleOptions) {
		options.TokenProvider = stscreds.StdinTokenProvider
	}))
	return config.LoadDefaultConfig(context.TODO(), loadOptions...)
}

// for CF interpolation and IAM Policy variables
func (*AWSService) escapeAwsInterpolation(str string) string {
	return awsVariable.ReplaceAllString(str, "$$$1")
}

// arnLastSegment returns the substring after the last occurrence of sep in s.
// Used to extract resource names from ARNs and URLs.
func arnLastSegment(s, sep string) string {
	parts := strings.Split(s, sep)
	return parts[len(parts)-1]
}

func (s *AWSService) getAccountNumber(config aws.Config) (*string, error) {
	stsSvc := sts.NewFromConfig(config)
	identity, err := stsSvc.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}
	return identity.Account, nil
}
