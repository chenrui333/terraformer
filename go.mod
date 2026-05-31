module github.com/chenrui333/terraformer

go 1.26.2

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/logging v1.18.0
	cloud.google.com/go/storage v1.62.2
	github.com/Azure/azure-storage-blob-go v0.15.0
	github.com/IBM-Cloud/bluemix-go v0.0.0-20260424100510-275dcc5549eb
	github.com/IBM/go-sdk-core/v3 v3.3.1
	github.com/IBM/go-sdk-core/v4 v4.10.0
	github.com/IBM/go-sdk-core/v5 v5.21.2
	github.com/IBM/ibm-cos-sdk-go v1.14.0
	github.com/IBM/keyprotect-go-client v0.16.0
	github.com/IBM/networking-go-sdk v0.53.4
	github.com/IBM/platform-services-go-sdk v0.97.4
	github.com/IBM/vpc-go-sdk v0.83.2
	github.com/OctopusDeploy/go-octopusdeploy v1.8.6
	github.com/PaloAltoNetworks/pango v0.10.2
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.107
	github.com/aliyun/aliyun-tablestore-go-sdk/v5 v5.0.6
	github.com/apache/openwhisk-client-go v0.0.0-20250309042127-fa7fa7e48863
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.32.18
	github.com/aws/aws-sdk-go-v2/credentials v1.19.17
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.48.0
	github.com/aws/aws-sdk-go-v2/service/acm v1.39.0
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.40.0
	github.com/aws/aws-sdk-go-v2/service/appsync v1.53.7
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.66.2
	github.com/aws/aws-sdk-go-v2/service/batch v1.64.3
	github.com/aws/aws-sdk-go-v2/service/budgets v1.43.6
	github.com/aws/aws-sdk-go-v2/service/cloud9 v1.34.0
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.11
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.64.0
	github.com/aws/aws-sdk-go-v2/service/cloudhsmv2 v1.34.25
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.55.11
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.57.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatchevents v1.32.25
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.74.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.16
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.33.14
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.35.15
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.23
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.33.24
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.60.2
	github.com/aws/aws-sdk-go-v2/service/configservice v1.62.3
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.63.0
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.30.22
	github.com/aws/aws-sdk-go-v2/service/devicefarm v1.38.10
	github.com/aws/aws-sdk-go-v2/service/docdb v1.48.15
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.4
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.304.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.57.2
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.38.15
	github.com/aws/aws-sdk-go-v2/service/ecs v1.80.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.16
	github.com/aws/aws-sdk-go-v2/service/eks v1.84.0
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.52.2
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.34.4
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.25
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.12
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.41.0
	github.com/aws/aws-sdk-go-v2/service/emr v1.59.2
	github.com/aws/aws-sdk-go-v2/service/firehose v1.42.16
	github.com/aws/aws-sdk-go-v2/service/glue v1.142.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.10
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.36.7
	github.com/aws/aws-sdk-go-v2/service/iot v1.73.0
	github.com/aws/aws-sdk-go-v2/service/kafka v1.52.0
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.43.7
	github.com/aws/aws-sdk-go-v2/service/kms v1.52.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.90.1
	github.com/aws/aws-sdk-go-v2/service/medialive v1.96.0
	github.com/aws/aws-sdk-go-v2/service/mediapackage v1.39.23
	github.com/aws/aws-sdk-go-v2/service/mediastore v1.29.23
	github.com/aws/aws-sdk-go-v2/service/memorydb v1.33.16
	github.com/aws/aws-sdk-go-v2/service/mq v1.34.22
	github.com/aws/aws-sdk-go-v2/service/neptune v1.44.5
	github.com/aws/aws-sdk-go-v2/service/opsworks v1.31.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.3
	github.com/aws/aws-sdk-go-v2/service/qldb v1.32.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.118.2
	github.com/aws/aws-sdk-go-v2/service/redshift v1.62.8
	github.com/aws/aws-sdk-go-v2/service/resourcegroups v1.33.26
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.101.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.7
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.71.0
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.39.15
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.24
	github.com/aws/aws-sdk-go-v2/service/sfn v1.41.0
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.17
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.27
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.6
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.39.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1
	github.com/aws/aws-sdk-go-v2/service/swf v1.33.18
	github.com/aws/aws-sdk-go-v2/service/waf v1.30.22
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.30.23
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.71.5
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.68.1
	github.com/aws/aws-sdk-go-v2/service/xray v1.36.23
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cloudflare/cloudflare-go v0.117.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/ddelnano/terraform-provider-mikrotik/client v0.0.0-20250110092516-5bc3b68c6245
	github.com/ddelnano/terraform-provider-xenorchestra/client v0.18.0-alpha1
	github.com/denverdino/aliyungo v0.0.0-20230411124812-ab98a9173ace
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/digitalocean/godo v1.192.0
	github.com/fastly/go-fastly/v7 v7.5.5
	github.com/google/go-github/v35 v35.3.0
	github.com/gophercloud/gophercloud v1.14.1
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/hashicorp/go-azure-helpers v0.79.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.8.0
	github.com/hashicorp/hcl v1.0.1-vault-7
	github.com/heimweh/go-pagerduty v0.0.0-20250801140645-0b96cfc9bf17
	github.com/heroku/heroku-go/v5 v5.5.0
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/honeycombio/terraform-provider-honeycombio v0.49.0
	github.com/iancoleman/strcase v0.3.0
	github.com/ionos-cloud/sdk-go-dbaas-mongo v1.4.1
	github.com/ionos-cloud/sdk-go-dbaas-postgres v1.1.4
	github.com/ionos-cloud/sdk-go/v6 v6.3.7
	github.com/jmespath/go-jmespath v0.4.0
	github.com/labd/commercetools-go-sdk v1.9.0
	github.com/linode/linodego v1.69.1
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/nicksnyder/go-i18n v1.10.3 // indirect
	github.com/okta/okta-sdk-golang/v2 v2.20.0
	github.com/opsgenie/opsgenie-go-sdk-v2 v1.2.23
	github.com/packethost/packngo v0.31.0
	github.com/pkg/errors v0.9.1
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/tencentyun/cos-go-sdk-v5 v0.7.73
	github.com/vultr/govultr v1.1.1
	github.com/yandex-cloud/go-genproto v0.82.0
	github.com/yandex-cloud/go-sdk v0.31.0
	github.com/zclconf/go-cty v1.18.1
	golang.org/x/oauth2 v0.36.0
	golang.org/x/text v0.37.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.280.0
	google.golang.org/genproto v0.0.0-20260504160031-60b97b32f348
	k8s.io/apimachinery v0.36.1
	k8s.io/client-go v0.36.1
)

require (
	github.com/IBM-Cloud/container-services-go-sdk v0.0.0-20250409011111-61af13302654
	github.com/IBM/go-sdk-core v1.1.0
	github.com/mackerelio/mackerel-client-go v0.42.0
	github.com/okta/terraform-provider-okta v0.0.0-20260507050055-d8f2f8783a1a
)

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.0 // indirect
	github.com/aws/smithy-go v1.25.1
	github.com/beevik/etree v1.6.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/crewjam/saml v0.5.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dghubble/sling v1.4.2 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.3+incompatible // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/errors v0.22.7 // indirect
	github.com/go-openapi/strfmt v0.26.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-resty/resty/v2 v2.17.2 // indirect
	github.com/go-routeros/routeros v0.0.0-20210123142807-2a44d57c6730 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/jsonapi v1.0.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.7 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-ieproxy v0.0.12 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/mozillazg/go-httpheader v0.4.0 // indirect
	github.com/oklog/run v1.2.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/peterhellberg/link v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russellhaering/goxmldsig v1.6.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/sourcegraph/jsonrpc2 v0.2.1 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20250811210735-e5fe3b51442e // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver v1.17.9 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sync v0.20.0
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
	gopkg.in/go-playground/validator.v9 v9.31.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.36.1
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/utils v0.0.0-20260319190234-28399d86e0b5 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/PuerkitoBio/rehttp v1.4.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/containerd/containerd v1.7.30 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-openapi/swag/cmdutils v0.26.0 // indirect
	github.com/go-openapi/swag/conv v0.26.0 // indirect
	github.com/go-openapi/swag/fileutils v0.26.0 // indirect
	github.com/go-openapi/swag/jsonname v0.26.0 // indirect
	github.com/go-openapi/swag/jsonutils v0.26.0 // indirect
	github.com/go-openapi/swag/loading v0.26.0 // indirect
	github.com/go-openapi/swag/mangling v0.26.0 // indirect
	github.com/go-openapi/swag/netutils v0.26.0 // indirect
	github.com/go-openapi/swag/stringutils v0.26.0 // indirect
	github.com/go-openapi/swag/typeutils v0.26.0 // indirect
	github.com/go-openapi/swag/yamlutils v0.26.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lib/pq v1.11.2 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rubenv/sql-migrate v1.8.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	k8s.io/apiextensions-apiserver v0.36.0 // indirect
	k8s.io/apiserver v0.36.0 // indirect
	k8s.io/component-base v0.36.0 // indirect
	k8s.io/kubectl v0.36.0 // indirect
	oras.land/oras-go/v2 v2.6.0 // indirect
	sigs.k8s.io/kustomize/api v0.21.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.21.1 // indirect
)

require (
	cloud.google.com/go/cloudbuild v1.30.0
	cloud.google.com/go/cloudtasks v1.18.0
	cloud.google.com/go/iam v1.11.0
	cloud.google.com/go/monitoring v1.29.0
	github.com/DataDog/datadog-api-client-go/v2 v2.60.0
	github.com/Myra-Security-GmbH/myrasec-go/v2 v2.51.0
	github.com/bradleyfalzon/ghinstallation/v2 v2.18.0
	github.com/manicminer/hamilton v0.72.0
	github.com/opalsecurity/opal-go v1.4.0
	gopkg.in/ns1/ns1-go.v2 v2.17.2
)

require (
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v0.12.0 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.56.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.56.0 // indirect
	github.com/Myra-Security-GmbH/signature v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/auth0/go-auth0 v1.42.0
	github.com/avast/retry-go v3.0.0+incompatible // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj v1.8.4 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-jose/go-jose/v3 v3.0.5 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag v0.26.0 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-github/v84 v84.0.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.15 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.2.0 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/jsonapi v1.5.0 // indirect
	github.com/hashicorp/terraform-plugin-log v0.10.0 // indirect
	github.com/lestrrat-go/backoff/v2 v2.0.8 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx v1.2.31 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/montanaflynn/stats v0.9.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/opentracing/opentracing-go v1.2.1-0.20220228012449-10b1cf09e00b // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/valyala/fastjson v1.6.10 // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
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
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260427160629-7cedc36a6bc4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20260501160325-927ab1f70cd6 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.0 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.1
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/analysisservices/armanalysisservices v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v4 v4.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6 v6.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2 v2.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v2 v2.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3 v3.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v9 v9.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2 v2.0.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6 v6.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/purview/armpurview v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v3 v3.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlocks v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2 v2.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity v0.14.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2 v2.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse v0.8.0
	github.com/IBM/continuous-delivery-go-sdk/v2 v2.0.8
	github.com/IBM/sarama v1.49.0
	github.com/aws/aws-msk-iam-sasl-signer-go v1.0.4
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.35.0
	github.com/aws/aws-sdk-go-v2/service/appconfig v1.44.0
	github.com/aws/aws-sdk-go-v2/service/appintegrations v1.37.9
	github.com/aws/aws-sdk-go-v2/service/appmesh v1.35.14
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.39.16
	github.com/aws/aws-sdk-go-v2/service/appstream v1.59.0
	github.com/aws/aws-sdk-go-v2/service/athena v1.57.6
	github.com/aws/aws-sdk-go-v2/service/backup v1.56.0
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.61.0
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.54.0
	github.com/aws/aws-sdk-go-v2/service/chatbot v1.14.23
	github.com/aws/aws-sdk-go-v2/service/chimesdkvoice v1.28.15
	github.com/aws/aws-sdk-go-v2/service/comprehend v1.41.0
	github.com/aws/aws-sdk-go-v2/service/connect v1.175.0
	github.com/aws/aws-sdk-go-v2/service/customerprofiles v1.61.0
	github.com/aws/aws-sdk-go-v2/service/detective v1.38.15
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.17
	github.com/aws/aws-sdk-go-v2/service/globalaccelerator v1.36.0
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.77.0
	github.com/aws/aws-sdk-go-v2/service/ivs v1.50.1
	github.com/aws/aws-sdk-go-v2/service/ivschat v1.21.22
	github.com/aws/aws-sdk-go-v2/service/kendra v1.60.23
	github.com/aws/aws-sdk-go-v2/service/lakeformation v1.47.8
	github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice v1.35.0
	github.com/aws/aws-sdk-go-v2/service/lexmodelsv2 v1.61.0
	github.com/aws/aws-sdk-go-v2/service/mediaconvert v1.91.2
	github.com/aws/aws-sdk-go-v2/service/mediapackagev2 v1.38.0
	github.com/aws/aws-sdk-go-v2/service/mwaa v1.40.1
	github.com/aws/aws-sdk-go-v2/service/networkmanager v1.42.0
	github.com/aws/aws-sdk-go-v2/service/notifications v1.7.22
	github.com/aws/aws-sdk-go-v2/service/notificationscontacts v1.6.0
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.69.0
	github.com/aws/aws-sdk-go-v2/service/opensearchserverless v1.30.3
	github.com/aws/aws-sdk-go-v2/service/pinpoint v1.39.23
	github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2 v1.28.2
	github.com/aws/aws-sdk-go-v2/service/pipes v1.23.22
	github.com/aws/aws-sdk-go-v2/service/quicksight v1.111.0
	github.com/aws/aws-sdk-go-v2/service/redshiftserverless v1.34.6
	github.com/aws/aws-sdk-go-v2/service/rekognition v1.51.24
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.44.0
	github.com/aws/aws-sdk-go-v2/service/s3control v1.70.1
	github.com/aws/aws-sdk-go-v2/service/s3tables v1.15.2
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.249.0
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.17.24
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.61.0
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.55.0
	github.com/aws/aws-sdk-go-v2/service/vpclattice v1.21.0
	github.com/gofrs/uuid/v3 v3.1.2
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/hashicorp/go-azure-sdk/sdk v0.20260520.1174751
	github.com/hashicorp/vault/api v1.23.0
	github.com/ionos-cloud/sdk-go-cert-manager v1.3.0
	github.com/ionos-cloud/sdk-go-container-registry v1.3.1
	github.com/ionos-cloud/sdk-go-dataplatform v1.1.1
	github.com/ionos-cloud/sdk-go-dns v1.4.0
	github.com/ionos-cloud/sdk-go-logging v1.3.0
	github.com/keycloak/terraform-provider-keycloak v0.0.0-20260508073653-4a34efa743f5
	github.com/launchdarkly/api-client-go/v22 v22.0.0
	github.com/logzio/logzio_terraform_client v1.30.2
	github.com/newrelic/newrelic-client-go v1.1.0
	github.com/newrelic/newrelic-client-go/v2 v2.86.1
	github.com/okta/okta-sdk-golang/v5 v5.0.6
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/as v1.3.88
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.3.102
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb v1.3.101
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn v1.3.90
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs v1.3.96
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.3.83
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.102
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.3.100
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod v1.3.78
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/es v1.3.95
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/gaap v1.3.34
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mongodb v1.3.93
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/pts v1.3.29
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/redis v1.3.79
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/scf v1.3.101
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ses v1.3.86
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl v1.3.94
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat v1.3.88
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tcaplusdb v1.3.17
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.3.83
	gitlab.com/gitlab-org/api/client-go v1.46.0
	helm.sh/helm/v3 v3.21.0
	k8s.io/cli-runtime v0.36.1
	software.sslmate.com/src/go-pkcs12 v0.7.1
)

replace gopkg.in/ns1/ns1-go.v2 => github.com/ns1/ns1-go/v2 v2.17.2

replace github.com/tencentcloud/tencentcloud-sdk-go => github.com/tencentcloud/tencentcloud-sdk-go v1.3.102

// Redirect stale transitive imports from abandoned dgrijalva/jwt-go to a compatible maintained fork.
replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.5+incompatible
