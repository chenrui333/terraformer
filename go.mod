module github.com/chenrui333/terraformer

go 1.26.2

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/logging v1.16.0
	cloud.google.com/go/storage v1.62.1
	github.com/Azure/azure-sdk-for-go v63.4.0+incompatible
	github.com/Azure/azure-storage-blob-go v0.15.0
	github.com/Azure/go-autorest/autorest v0.11.30
	github.com/IBM-Cloud/bluemix-go v0.0.0-20220624043500-d538cb4fd9be
	github.com/IBM/go-sdk-core/v3 v3.3.1
	github.com/IBM/go-sdk-core/v4 v4.9.0
	github.com/IBM/go-sdk-core/v5 v5.18.1
	github.com/IBM/ibm-cos-sdk-go v1.5.0
	github.com/IBM/keyprotect-go-client v0.8.1
	github.com/IBM/networking-go-sdk v0.30.0
	github.com/IBM/platform-services-go-sdk v0.26.1
	github.com/IBM/vpc-go-sdk v0.64.0
	github.com/OctopusDeploy/go-octopusdeploy v1.6.0
	github.com/PaloAltoNetworks/pango v0.8.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1247
	github.com/aliyun/aliyun-tablestore-go-sdk v4.1.2+incompatible
	github.com/apache/openwhisk-client-go v0.0.0-20210106144548-17d556327cd3
	github.com/aws/aws-sdk-go-v2 v1.41.6
	github.com/aws/aws-sdk-go-v2/config v1.32.16
	github.com/aws/aws-sdk-go-v2/credentials v1.19.15
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.47.1
	github.com/aws/aws-sdk-go-v2/service/acm v1.38.2
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.39.2
	github.com/aws/aws-sdk-go-v2/service/appsync v1.53.6
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.66.1
	github.com/aws/aws-sdk-go-v2/service/batch v1.64.0
	github.com/aws/aws-sdk-go-v2/service/budgets v1.43.5
	github.com/aws/aws-sdk-go-v2/service/cloud9 v1.33.21
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.10
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.61.1
	github.com/aws/aws-sdk-go-v2/service/cloudhsmv2 v1.34.24
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.55.10
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.56.2
	github.com/aws/aws-sdk-go-v2/service/cloudwatchevents v1.32.24
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.70.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.14
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.33.13
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.35.14
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.22
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.33.23
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.60.1
	github.com/aws/aws-sdk-go-v2/service/configservice v1.62.2
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.30.21
	github.com/aws/aws-sdk-go-v2/service/devicefarm v1.38.9
	github.com/aws/aws-sdk-go-v2/service/docdb v1.48.14
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.2
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.299.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.57.1
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.38.14
	github.com/aws/aws-sdk-go-v2/service/ecs v1.79.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.15
	github.com/aws/aws-sdk-go-v2/service/eks v1.82.1
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.52.1
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.34.3
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.24
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.11
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.40.1
	github.com/aws/aws-sdk-go-v2/service/emr v1.59.1
	github.com/aws/aws-sdk-go-v2/service/firehose v1.42.14
	github.com/aws/aws-sdk-go-v2/service/glue v1.139.3
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.8
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.36.6
	github.com/aws/aws-sdk-go-v2/service/iot v1.72.7
	github.com/aws/aws-sdk-go-v2/service/kafka v1.50.0
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.43.6
	github.com/aws/aws-sdk-go-v2/service/kms v1.50.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.90.0
	github.com/aws/aws-sdk-go-v2/service/medialive v1.95.1
	github.com/aws/aws-sdk-go-v2/service/mediapackage v1.39.22
	github.com/aws/aws-sdk-go-v2/service/mediastore v1.29.22
	github.com/aws/aws-sdk-go-v2/service/mq v1.34.20
	github.com/aws/aws-sdk-go-v2/service/opsworks v1.31.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.2
	github.com/aws/aws-sdk-go-v2/service/qldb v1.32.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.118.1
	github.com/aws/aws-sdk-go-v2/service/redshift v1.62.6
	github.com/aws/aws-sdk-go-v2/service/resourcegroups v1.33.25
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.6
	github.com/aws/aws-sdk-go-v2/service/s3 v1.100.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.6
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.69.1
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.39.13
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.23
	github.com/aws/aws-sdk-go-v2/service/sfn v1.40.11
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.16
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.26
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.5
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.37.7
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.0
	github.com/aws/aws-sdk-go-v2/service/swf v1.33.17
	github.com/aws/aws-sdk-go-v2/service/waf v1.30.21
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.30.22
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.71.4
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.67.3
	github.com/aws/aws-sdk-go-v2/service/xray v1.36.22
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cloudflare/cloudflare-go v0.13.6
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/ddelnano/terraform-provider-mikrotik/client v0.0.0-20210401060029-7f652169b2c4
	github.com/ddelnano/terraform-provider-xenorchestra/client v0.0.0-20210401070256-0d721c6762ef
	github.com/denverdino/aliyungo v0.0.0-20200327235253-d59c209c7e93
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/digitalocean/godo v1.187.0
	github.com/fastly/go-fastly/v7 v7.5.5
	github.com/google/go-github/v35 v35.3.0
	github.com/gophercloud/gophercloud v1.14.1
	github.com/grafana/grafana-api-golang-client v0.0.0-20210218192924-9ccd2365d2a6
	github.com/hashicorp/go-azure-helpers v0.48.0
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.4.4
	github.com/hashicorp/hcl v1.0.1-vault-7
	github.com/hashicorp/terraform v0.12.31
	github.com/heimweh/go-pagerduty v0.0.0-20210930203304-530eff2acdc6
	github.com/heroku/heroku-go/v5 v5.5.0
	github.com/hokaccha/go-prettyjson v0.0.0-20210113012101-fb4e108d2519 // indirect
	github.com/honeycombio/terraform-provider-honeycombio v0.10.0
	github.com/iancoleman/strcase v0.3.0
	github.com/ionos-cloud/sdk-go-dbaas-mongo v1.4.1
	github.com/ionos-cloud/sdk-go-dbaas-postgres v1.1.4
	github.com/ionos-cloud/sdk-go/v6 v6.3.7
	github.com/jmespath/go-jmespath v0.4.0
	github.com/jonboydell/logzio_client v1.2.0
	github.com/labd/commercetools-go-sdk v0.3.1
	github.com/linode/linodego v1.68.0
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/mrparkers/terraform-provider-keycloak v0.0.0-20221013232944-56f37a07590d
	github.com/nicksnyder/go-i18n v1.10.1 // indirect
	github.com/okta/okta-sdk-golang/v2 v2.20.0
	github.com/opsgenie/opsgenie-go-sdk-v2 v1.2.23
	github.com/packethost/packngo v0.31.0
	github.com/pkg/errors v0.9.1
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/tencentyun/cos-go-sdk-v5 v0.7.34
	github.com/vultr/govultr v1.1.1
	github.com/xanzy/go-gitlab v0.50.2
	github.com/yandex-cloud/go-genproto v0.0.0-20220314102905-1acaee8ca7eb
	github.com/yandex-cloud/go-sdk v0.0.0-20220314105123-d0c2a928feb6
	github.com/zclconf/go-cty v1.11.0
	github.com/zorkian/go-datadog-api v2.30.0+incompatible
	golang.org/x/oauth2 v0.36.0
	golang.org/x/text v0.36.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.276.0
	google.golang.org/genproto v0.0.0-20260420184626-e10c466a9529
	k8s.io/apimachinery v0.36.0
	k8s.io/client-go v0.36.0
)

require (
	github.com/IBM-Cloud/container-services-go-sdk v0.0.0-20210705152127-41ca00fc9a62
	github.com/IBM/go-sdk-core v1.1.0
	github.com/hashicorp/terraform-svchost v0.0.0-20200729002733-f050f53b9734 // indirect
	github.com/mackerelio/mackerel-client-go v0.41.0
	github.com/okta/terraform-provider-okta v0.0.0-20210924173942-a5a664459d3b
	github.com/zclconf/go-cty-yaml v1.0.2 // indirect
)

require (
	github.com/antihax/optional v1.0.0 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/launchdarkly/api-client-go v5.3.0+incompatible
)

require github.com/newrelic/newrelic-client-go v1.1.0

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.7 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.1 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.1 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.2 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/appscode/go-querystring v0.0.0-20170504095604-0126cfb3f1dc // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.9 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.20 // indirect
	github.com/aws/smithy-go v1.25.1
	github.com/beevik/etree v1.6.0 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/bgentry/speakeasy v0.1.0 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/bmatcuk/doublestar v1.1.5 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/crewjam/saml v0.4.14 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dghubble/sling v1.4.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-resty/resty/v2 v2.17.2 // indirect
	github.com/go-routeros/routeros v0.0.0-20210123142807-2a44d57c6730 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/jsonapi v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/gax-go/v2 v2.21.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-getter v1.8.6 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/hcl/v2 v2.14.0 // indirect
	github.com/hashicorp/hil v0.0.0-20190212112733-ab17b08d6590 // indirect
	github.com/hashicorp/yamux v0.0.0-20211028200310-0bc27b27de87 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/cli v1.1.5 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/hashstructure v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mozillazg/go-httpheader v0.2.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/patrickmn/go-cache v0.0.0-20180815053127-5633e0862627 // indirect
	github.com/pborman/uuid v1.2.0 // indirect
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/peterhellberg/link v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/posener/complete v1.2.1 // indirect
	github.com/russellhaering/goxmldsig v1.6.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/jsonrpc2 v0.0.0-20210201082850-366fbb520750 // indirect
	github.com/spf13/afero v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.12 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/grpc v1.80.0 // indirect
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af // indirect
	gopkg.in/auth0.v5 v5.21.1
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.36.0 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

require github.com/PuerkitoBio/rehttp v1.0.0 // indirect

require (
	cloud.google.com/go/cloudbuild v1.28.0
	cloud.google.com/go/cloudtasks v1.16.0
	cloud.google.com/go/iam v1.9.0
	cloud.google.com/go/monitoring v1.27.0
	github.com/DataDog/datadog-api-client-go/v2 v2.59.0
	github.com/Myra-Security-GmbH/myrasec-go/v2 v2.28.0
	github.com/bradleyfalzon/ghinstallation/v2 v2.18.0
	github.com/manicminer/hamilton v0.50.0
	github.com/opalsecurity/opal-go v1.0.19
	gopkg.in/ns1/ns1-go.v2 v2.6.5
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v0.11.0 // indirect
	github.com/DataDog/zstd v1.5.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.1 // indirect
	github.com/Myra-Security-GmbH/signature v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.10 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.36.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-jose/go-jose/v3 v3.0.5 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/validator/v10 v10.19.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-github/v84 v84.0.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.14 // indirect
	github.com/hashicorp/aws-sdk-go-base/v2 v2.0.0-beta.72 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/terraform-plugin-log v0.10.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx v1.2.29 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/manicminer/hamilton-autorest v0.2.0 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/valyala/fastjson v1.6.3 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.43.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	k8s.io/kube-openapi v0.0.0-20260317180543-43fb72c5454a // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
)

require (
	github.com/IBM/continuous-delivery-go-sdk/v2 v2.0.2
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.34.2
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.16
	github.com/gofrs/uuid/v3 v3.1.2
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/hashicorp/vault/api v1.23.0
	github.com/ionos-cloud/sdk-go-cert-manager v1.3.0
	github.com/ionos-cloud/sdk-go-container-registry v1.3.1
	github.com/ionos-cloud/sdk-go-dataplatform v1.1.1
	github.com/ionos-cloud/sdk-go-dns v1.4.0
	github.com/ionos-cloud/sdk-go-logging v1.3.0
	github.com/okta/okta-sdk-golang/v5 v5.0.6
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/as v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.0.694
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/es v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/gaap v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mongodb v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/pts v1.0.694
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/redis v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/scf v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ses v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tcaplusdb v1.0.392
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.0.392
)

replace gopkg.in/jarcoal/httpmock.v1 => github.com/jarcoal/httpmock v1.0.5

replace gopkg.in/ns1/ns1-go.v2 => github.com/ns1/ns1-go/v2 v2.6.5

replace github.com/tencentcloud/tencentcloud-sdk-go => github.com/tencentcloud/tencentcloud-sdk-go v1.0.392

// Redirect stale transitive imports from abandoned dgrijalva/jwt-go to a compatible maintained fork.
replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.5+incompatible
