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

* KAFKA_BOOTSTRAP_SERVERS
* KAFKA_VERSION
* KAFKA_ENABLE_TLS
* KAFKA_SKIP_VERIFY
* KAFKA_CA_CERT
* KAFKA_CLIENT_CERT
* KAFKA_CLIENT_KEY
* KAFKA_CLIENT_KEY_PASSPHRASE
* KAFKA_SASL_MECHANISM
* KAFKA_SASL_USERNAME
* KAFKA_SASL_PASSWORD
* KAFKA_SASL_IAM_AWS_REGION
* AWS_PROFILE
* AWS_ROLE_ARN
* AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE
* AWS_CONTAINER_CREDENTIALS_FULL_URI
* AWS_SHARED_CONFIG_FILES
* KAFKA_SASL_TOKEN_URL
* KAFKA_SASL_OAUTH_SCOPES

OAuthBearer imports use KAFKA_SASL_TOKEN_URL with KAFKA_SASL_USERNAME, KAFKA_SASL_PASSWORD, and optional KAFKA_SASL_OAUTH_SCOPES.

Terraformer intentionally does not write generated HCL containing SASL passwords, TLS private keys, AWS access keys, AWS secret keys, AWS session tokens, OAuth tokens, or SCRAM passwords.

List of supported Kafka services:

* topics
    * kafka_topic
