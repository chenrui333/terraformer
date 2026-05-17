// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/endpointcreds"
	"github.com/xdg-go/scram"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultKafkaVersion = "2.7.0"
	defaultKafkaTimeout = 120
)

type Config struct {
	BootstrapServers                       []string
	KafkaVersion                           string
	TLSEnabled                             bool
	SkipTLSVerify                          bool
	SASLMechanism                          string
	SASLUsername                           string
	SASLAWSRegion                          string
	SASLAWSContainerAuthorizationTokenFile string
	SASLAWSContainerCredentialsFullURI     string
	SASLAWSRoleARN                         string
	SASLAWSExternalID                      string
	SASLAWSProfile                         string
	SASLAWSSharedConfigFiles               []string
	SASLTokenURL                           string
	SASLOAuthScopes                        []string
	CACert                                 string
	ClientCert                             string
	Timeout                                int

	ClientKey           string `json:"-"`
	ClientKeyPassphrase string `json:"-"`
	SASLPassword        string `json:"-"`
	SASLAWSAccessKey    string `json:"-"`
	SASLAWSSecretKey    string `json:"-"`
	SASLAWSSessionToken string `json:"-"`
	SASLAWSCredsDebug   bool   `json:"-"`
}

func ConfigFromEnv() Config {
	return Config{
		BootstrapServers:                       splitCSV(os.Getenv("KAFKA_BOOTSTRAP_SERVERS")),
		KafkaVersion:                           envString("KAFKA_VERSION", defaultKafkaVersion),
		TLSEnabled:                             envBool("KAFKA_ENABLE_TLS", true),
		SkipTLSVerify:                          envBool("KAFKA_SKIP_VERIFY", false),
		SASLMechanism:                          envString("KAFKA_SASL_MECHANISM", "plain"),
		SASLUsername:                           os.Getenv("KAFKA_SASL_USERNAME"),
		SASLAWSRegion:                          firstNonEmpty(os.Getenv("KAFKA_SASL_IAM_AWS_REGION"), os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION")),
		SASLAWSContainerAuthorizationTokenFile: os.Getenv("AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE"),
		SASLAWSContainerCredentialsFullURI:     os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI"),
		SASLAWSRoleARN:                         os.Getenv("AWS_ROLE_ARN"),
		SASLAWSExternalID:                      os.Getenv("KAFKA_SASL_AWS_EXTERNAL_ID"),
		SASLAWSProfile:                         os.Getenv("AWS_PROFILE"),
		SASLAWSSharedConfigFiles:               splitCSV(os.Getenv("AWS_SHARED_CONFIG_FILES")),
		SASLTokenURL:                           firstNonEmpty(os.Getenv("KAFKA_SASL_TOKEN_URL"), os.Getenv("TOKEN_URL")),
		SASLOAuthScopes:                        splitCSV(os.Getenv("KAFKA_SASL_OAUTH_SCOPES")),
		CACert:                                 os.Getenv("KAFKA_CA_CERT"),
		ClientCert:                             os.Getenv("KAFKA_CLIENT_CERT"),
		Timeout:                                envInt("KAFKA_TIMEOUT", defaultKafkaTimeout),
		ClientKey:                              os.Getenv("KAFKA_CLIENT_KEY"),
		ClientKeyPassphrase:                    os.Getenv("KAFKA_CLIENT_KEY_PASSPHRASE"),
		SASLPassword:                           os.Getenv("KAFKA_SASL_PASSWORD"),
		SASLAWSAccessKey:                       os.Getenv("AWS_ACCESS_KEY_ID"),
		SASLAWSSecretKey:                       os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SASLAWSSessionToken:                    os.Getenv("AWS_SESSION_TOKEN"),
		SASLAWSCredsDebug:                      envBool("AWS_CREDS_DEBUG", false),
	}
}

func EncodeConfig(config Config) string {
	encoded, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"BootstrapServers":                       c.BootstrapServers,
		"KafkaVersion":                           c.KafkaVersion,
		"TLSEnabled":                             c.TLSEnabled,
		"SkipTLSVerify":                          c.SkipTLSVerify,
		"SASLMechanism":                          c.SASLMechanism,
		"SASLUsername":                           c.SASLUsername,
		"SASLAWSRegion":                          c.SASLAWSRegion,
		"SASLAWSContainerAuthorizationTokenFile": c.SASLAWSContainerAuthorizationTokenFile,
		"SASLAWSContainerCredentialsFullURI":     c.SASLAWSContainerCredentialsFullURI,
		"SASLAWSRoleARN":                         c.SASLAWSRoleARN,
		"SASLAWSExternalID":                      c.SASLAWSExternalID,
		"SASLAWSProfile":                         c.SASLAWSProfile,
		"SASLAWSSharedConfigFiles":               c.SASLAWSSharedConfigFiles,
		"SASLTokenURL":                           c.SASLTokenURL,
		"SASLOAuthScopes":                        c.SASLOAuthScopes,
		"CACert":                                 c.CACert,
		"ClientCert":                             c.ClientCert,
		"Timeout":                                c.Timeout,
	})
}

func configFromArgs(args []string) (Config, error) {
	config := ConfigFromEnv()
	if len(args) == 0 || args[0] == "" {
		return config, nil
	}
	if err := json.Unmarshal([]byte(args[0]), &config); err != nil {
		return Config{}, fmt.Errorf("kafka: decode provider config: %w", err)
	}
	config.applyEnvSecrets()
	return config, nil
}

func (c *Config) applyEnvSecrets() {
	envConfig := ConfigFromEnv()
	if c.ClientKey == "" {
		c.ClientKey = envConfig.ClientKey
	}
	if c.ClientKeyPassphrase == "" {
		c.ClientKeyPassphrase = envConfig.ClientKeyPassphrase
	}
	if c.SASLPassword == "" {
		c.SASLPassword = envConfig.SASLPassword
	}
	if c.SASLAWSAccessKey == "" {
		c.SASLAWSAccessKey = envConfig.SASLAWSAccessKey
	}
	if c.SASLAWSSecretKey == "" {
		c.SASLAWSSecretKey = envConfig.SASLAWSSecretKey
	}
	if c.SASLAWSSessionToken == "" {
		c.SASLAWSSessionToken = envConfig.SASLAWSSessionToken
	}
	if !c.SASLAWSCredsDebug {
		c.SASLAWSCredsDebug = envConfig.SASLAWSCredsDebug
	}
}

func (c Config) validate() error {
	if len(c.BootstrapServers) == 0 {
		return errors.New("kafka: bootstrap servers are required via --bootstrap-servers or KAFKA_BOOTSTRAP_SERVERS")
	}
	if c.KafkaVersion == "" {
		return errors.New("kafka: kafka version is required")
	}
	if c.Timeout <= 0 {
		return errors.New("kafka: timeout must be positive")
	}
	return nil
}

func (c Config) newSaramaConfig() (*sarama.Config, error) {
	config := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion(c.KafkaVersion)
	if err != nil {
		return nil, fmt.Errorf("kafka: parse kafka version %q: %w", c.KafkaVersion, err)
	}
	config.Version = version
	config.ClientID = "terraformer-kafka"
	config.Admin.Timeout = time.Duration(c.Timeout) * time.Second
	config.Metadata.Full = true
	config.Metadata.AllowAutoTopicCreation = false
	config.Metadata.Timeout = time.Duration(c.Timeout) * time.Second
	config.Net.ReadTimeout = time.Duration(c.Timeout) * time.Second
	config.Net.WriteTimeout = time.Duration(c.Timeout) * time.Second

	if c.saslEnabled() {
		if err := c.configureSASL(config); err != nil {
			return nil, err
		}
	}
	if c.TLSEnabled {
		tlsConfig, err := newTLSConfig(c.ClientCert, c.ClientKey, c.CACert, c.ClientKeyPassphrase)
		if err != nil {
			return nil, err
		}
		tlsConfig.InsecureSkipVerify = c.SkipTLSVerify
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}
	return config, nil
}

func (c Config) saslEnabled() bool {
	return c.SASLMechanism == "aws-iam" ||
		c.SASLMechanism == "oauthbearer" ||
		c.SASLMechanism == "scram-sha256" ||
		c.SASLMechanism == "scram-sha512" ||
		c.SASLUsername != "" ||
		c.SASLPassword != ""
}

func (c Config) configureSASL(config *sarama.Config) error {
	mechanism := c.SASLMechanism
	if mechanism == "" {
		mechanism = "plain"
	}
	config.Net.SASL.Enable = true
	config.Net.SASL.Handshake = true
	config.Net.SASL.User = c.SASLUsername
	config.Net.SASL.Password = c.SASLPassword

	switch mechanism {
	case "plain":
		if err := c.requireSASLCredentials(mechanism); err != nil {
			return err
		}
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	case "scram-sha256":
		if err := c.requireSASLCredentials(mechanism); err != nil {
			return err
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &xdgSCRAMClient{HashGeneratorFcn: scram.SHA256}
		}
	case "scram-sha512":
		if err := c.requireSASLCredentials(mechanism); err != nil {
			return err
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &xdgSCRAMClient{HashGeneratorFcn: scram.SHA512}
		}
	case "aws-iam":
		if c.SASLAWSRegion == "" {
			return errors.New("kafka: sasl aws region is required for aws-iam authentication")
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.SASL.TokenProvider = awsIAMTokenProvider{config: c}
	case "oauthbearer":
		tokenProvider, err := c.oauthBearerTokenProvider()
		if err != nil {
			return err
		}
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.SASL.TokenProvider = tokenProvider
	default:
		return fmt.Errorf("kafka: unsupported sasl mechanism %q", mechanism)
	}
	return nil
}

func (c Config) requireSASLCredentials(mechanism string) error {
	if c.SASLUsername == "" || c.SASLPassword == "" {
		return fmt.Errorf("kafka: sasl username and password are required for %s authentication", mechanism)
	}
	return nil
}

func (c Config) oauthBearerTokenProvider() (sarama.AccessTokenProvider, error) {
	if c.SASLTokenURL == "" {
		return nil, errors.New("kafka: KAFKA_SASL_TOKEN_URL is required for oauthbearer authentication")
	}
	if c.SASLUsername == "" {
		return nil, errors.New("kafka: sasl username is required for oauthbearer token URL authentication")
	}
	if c.SASLPassword == "" {
		return nil, errors.New("kafka: KAFKA_SASL_PASSWORD is required for oauthbearer token URL authentication")
	}

	client := &http.Client{Timeout: time.Duration(c.Timeout) * time.Second}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	source := (&clientcredentials.Config{
		ClientID:     c.SASLUsername,
		ClientSecret: c.SASLPassword,
		TokenURL:     c.SASLTokenURL,
		Scopes:       c.SASLOAuthScopes,
	}).TokenSource(ctx)
	return oauthBearerTokenProvider{tokenSource: source}, nil
}

type xdgSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

func (x *xdgSCRAMClient) Begin(userName, password, authzID string) error {
	client, err := x.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.Client = client
	x.ClientConversation = client.NewConversation()
	return nil
}

func (x *xdgSCRAMClient) Step(challenge string) (string, error) {
	return x.ClientConversation.Step(challenge)
}

func (x *xdgSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

type awsIAMTokenProvider struct {
	config Config
}

func (p awsIAMTokenProvider) Token() (*sarama.AccessToken, error) {
	signer.AwsDebugCreds = p.config.SASLAWSCredsDebug
	var token string
	var err error
	switch {
	case p.config.SASLAWSContainerAuthorizationTokenFile != "" && p.config.SASLAWSContainerCredentialsFullURI != "":
		var authorizationToken []byte
		authorizationToken, err = os.ReadFile(p.config.SASLAWSContainerAuthorizationTokenFile)
		if err != nil {
			return nil, fmt.Errorf("kafka: read AWS container authorization token file: %w", err)
		}
		credProvider := endpointcreds.New(p.config.SASLAWSContainerCredentialsFullURI, func(o *endpointcreds.Options) {
			o.AuthorizationToken = string(authorizationToken)
		})
		token, _, err = signer.GenerateAuthTokenFromCredentialsProvider(context.TODO(), p.config.SASLAWSRegion, credProvider)
	case p.config.SASLAWSRoleARN != "":
		token, _, err = signer.GenerateAuthTokenFromRoleWithExternalId(context.TODO(), p.config.SASLAWSRegion, p.config.SASLAWSRoleARN, "terraformer-kafka", p.config.SASLAWSExternalID)
	case p.config.SASLAWSProfile != "":
		if len(p.config.SASLAWSSharedConfigFiles) > 0 {
			token, _, err = signer.GenerateAuthTokenFromProfileWithSharedConfigFiles(context.TODO(), p.config.SASLAWSRegion, p.config.SASLAWSProfile, p.config.SASLAWSSharedConfigFiles)
		} else {
			token, _, err = signer.GenerateAuthTokenFromProfile(context.TODO(), p.config.SASLAWSRegion, p.config.SASLAWSProfile)
		}
	case p.config.SASLAWSAccessKey != "" && p.config.SASLAWSSecretKey != "":
		token, _, err = signer.GenerateAuthTokenFromCredentialsProvider(
			context.TODO(),
			p.config.SASLAWSRegion,
			credentials.NewStaticCredentialsProvider(p.config.SASLAWSAccessKey, p.config.SASLAWSSecretKey, p.config.SASLAWSSessionToken),
		)
	default:
		token, _, err = signer.GenerateAuthToken(context.TODO(), p.config.SASLAWSRegion)
	}
	if err != nil {
		return nil, err
	}
	return &sarama.AccessToken{Token: token}, nil
}

type oauthBearerTokenProvider struct {
	tokenSource oauth2.TokenSource
}

func (p oauthBearerTokenProvider) Token() (*sarama.AccessToken, error) {
	if p.tokenSource == nil {
		return nil, errors.New("kafka: oauthbearer token provider is not configured")
	}
	token, err := p.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("kafka: fetch oauthbearer token: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("kafka: oauthbearer token response did not include access_token")
	}
	return &sarama.AccessToken{Token: token.AccessToken}, nil
}

func newTLSConfig(clientCert, clientKey, caCert, clientKeyPassphrase string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if (clientCert == "") != (clientKey == "") {
		return nil, errors.New("kafka: client certificate and client key must be provided together")
	}
	if clientCert != "" && clientKey != "" {
		certBytes, err := pemBytes(clientCert)
		if err != nil {
			return nil, err
		}
		keyBlock, keyBytes, err := pemBlockAndBytes(clientKey)
		if err != nil {
			return nil, err
		}
		if x509.IsEncryptedPEMBlock(keyBlock) { //nolint:staticcheck // Legacy encrypted PEM keys are supported for Kafka provider compatibility.
			decrypted, err := x509.DecryptPEMBlock(keyBlock, []byte(clientKeyPassphrase)) //nolint:staticcheck // Legacy encrypted PEM keys are supported for Kafka provider compatibility.
			if err != nil {
				return nil, err
			}
			keyBytes = pem.EncodeToMemory(&pem.Block{Type: keyBlock.Type, Bytes: decrypted})
		}
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	if caCert == "" {
		return tlsConfig, nil
	}
	caBytes, err := pemBytes(caCert)
	if err != nil {
		return nil, err
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, errors.New("kafka: could not add CA certificate")
	}
	tlsConfig.RootCAs = pool
	return tlsConfig, nil
}

func pemBytes(input string) ([]byte, error) {
	_, bytes, err := pemBlockAndBytes(input)
	return bytes, err
}

func pemBlockAndBytes(input string) (*pem.Block, []byte, error) {
	bytes := []byte(input)
	block, _ := pem.Decode(bytes)
	if block != nil {
		return block, bytes, nil
	}
	loaded, err := os.ReadFile(input)
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(loaded)
	if block == nil {
		return nil, nil, errors.New("kafka: unable to decode PEM data")
	}
	return block, loaded, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
