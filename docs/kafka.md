### Use with Kafka

Example:

    export KAFKA_BOOTSTRAP_SERVERS=broker1.example.com:9092,broker2.example.com:9092
    export KAFKA_VERSION=2.7.0

    terraformer import kafka --resources=topics
    terraformer import kafka --resources=topics --filter=topic=orders:payments.events

The Kafka provider also accepts bootstrap servers through the CLI:

    terraformer import kafka --resources=topics --bootstrap-servers=broker1.example.com:9092,broker2.example.com:9092

Authentication and TLS configuration should use environment variables where secrets are involved. Non-secret settings are also available as CLI flags.

Common environment variables:

* KAFKA_BOOTSTRAP_SERVERS: required unless `--bootstrap-servers` is set; comma-separated Kafka broker `host:port` values.
* KAFKA_VERSION: optional Kafka protocol version, for example `2.7.0`.
* KAFKA_ENABLE_TLS: optional boolean; defaults to `true`.
* KAFKA_SKIP_VERIFY: optional boolean for skipping TLS certificate verification; defaults to `false`.
* KAFKA_CA_CERT: optional CA certificate PEM or file path.
* KAFKA_CLIENT_CERT: optional mTLS client certificate PEM or file path; must be paired with `KAFKA_CLIENT_KEY`.
* KAFKA_CLIENT_KEY: optional mTLS client private key PEM or file path; must be paired with `KAFKA_CLIENT_CERT`.
* KAFKA_CLIENT_KEY_PASSPHRASE: optional passphrase for encrypted legacy PEM client keys.
* KAFKA_SASL_MECHANISM: optional SASL mechanism, such as `plain`, `scram-sha256`, `scram-sha512`, `aws-iam`, or `oauthbearer`.
* KAFKA_SASL_USERNAME: required for `plain`, `scram-*`, and OAuth token URL authentication.
* KAFKA_SASL_PASSWORD: required for `plain`, `scram-*`, and OAuth token URL authentication.
* KAFKA_SASL_IAM_AWS_REGION: required for `aws-iam` unless `AWS_REGION` or `AWS_DEFAULT_REGION` is set.
* KAFKA_SASL_AWS_EXTERNAL_ID: optional external ID when assuming an AWS IAM role for `aws-iam`.
* KAFKA_SASL_TOKEN_URL: required for `oauthbearer`; OAuth token endpoint URL.
* KAFKA_SASL_OAUTH_SCOPES: optional comma-separated OAuth scopes.
* TOKEN_URL: optional alias for `KAFKA_SASL_TOKEN_URL`.
* AWS_PROFILE: optional AWS profile for `aws-iam`.
* AWS_REGION: optional AWS region fallback for `aws-iam`.
* AWS_DEFAULT_REGION: optional AWS region fallback for `aws-iam`.
* AWS_ROLE_ARN: optional AWS role ARN for `aws-iam`.
* AWS_ACCESS_KEY_ID: optional AWS access key ID for `aws-iam`; use environment variables rather than generated HCL.
* AWS_SECRET_ACCESS_KEY: optional AWS secret access key for `aws-iam`; use environment variables rather than generated HCL.
* AWS_SESSION_TOKEN: optional AWS session token for `aws-iam`; use environment variables rather than generated HCL.
* AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE: optional ECS container authorization token file path for `aws-iam`.
* AWS_CONTAINER_CREDENTIALS_FULL_URI: optional ECS container credentials URI for `aws-iam`.
* AWS_SHARED_CONFIG_FILES: optional comma-separated AWS shared config file paths.

OAuthBearer imports use KAFKA_SASL_TOKEN_URL with KAFKA_SASL_USERNAME, KAFKA_SASL_PASSWORD, and optional KAFKA_SASL_OAUTH_SCOPES.

Terraformer intentionally does not write generated HCL containing SASL passwords, TLS private keys, AWS access keys, AWS secret keys, AWS session tokens, OAuth tokens, or SCRAM passwords.

List of supported Kafka services:

* topics
    * kafka_topic

Unsupported Kafka resources with evidence-backed import limitations are tracked in [unsupported_resources.json](../providers/kafka/unsupported_resources.json). Terraformer does not emit `kafka_quota` because Mongey/kafka v0.13.1 documents an import form but does not expose a resource importer, and it does not emit `kafka_user_scram_credential` because refresh cannot recover the required password material that Terraformer must not export or synthesize.
