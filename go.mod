module github.com/chenrui333/terraformer

go 1.26.4

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/logging v1.18.0
	cloud.google.com/go/storage v1.62.3
	github.com/Azure/azure-storage-blob-go v0.15.0
	github.com/IBM-Cloud/bluemix-go v0.0.0-20260605140443-e006e534aa6d
	github.com/IBM/go-sdk-core/v5 v5.21.4
	github.com/IBM/ibm-cos-sdk-go v1.14.1
	github.com/IBM/keyprotect-go-client v0.16.0
	github.com/IBM/networking-go-sdk v0.53.5
	github.com/IBM/platform-services-go-sdk v0.99.1
	github.com/IBM/vpc-go-sdk v0.84.0
	github.com/OctopusDeploy/go-octopusdeploy/v2 v2.109.0
	github.com/PaloAltoNetworks/pango v0.10.2
	github.com/aliyun/alibaba-cloud-sdk-go v1.63.107
	github.com/aliyun/aliyun-tablestore-go-sdk/v5 v5.0.6
	github.com/apache/openwhisk-client-go v0.0.0-20250309042127-fa7fa7e48863
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/service/accessanalyzer v1.49.5
	github.com/aws/aws-sdk-go-v2/service/acm v1.39.6
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.40.6
	github.com/aws/aws-sdk-go-v2/service/appsync v1.54.4
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.67.4
	github.com/aws/aws-sdk-go-v2/service/batch v1.65.6
	github.com/aws/aws-sdk-go-v2/service/budgets v1.44.6
	github.com/aws/aws-sdk-go-v2/service/cloud9 v1.34.6
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.72.1
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.65.2
	github.com/aws/aws-sdk-go-v2/service/cloudhsmv2 v1.35.4
	github.com/aws/aws-sdk-go-v2/service/cloudtrail v1.56.4
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.58.3
	github.com/aws/aws-sdk-go-v2/service/cloudwatchevents v1.33.5
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.75.2
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.69.4
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.34.4
	github.com/aws/aws-sdk-go-v2/service/codedeploy v1.36.4
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.47.4
	github.com/aws/aws-sdk-go-v2/service/cognitoidentity v1.34.4
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.61.4
	github.com/aws/aws-sdk-go-v2/service/configservice v1.64.1
	github.com/aws/aws-sdk-go-v2/service/databasemigrationservice v1.64.4
	github.com/aws/aws-sdk-go-v2/service/datapipeline v1.31.5
	github.com/aws/aws-sdk-go-v2/service/devicefarm v1.39.5
	github.com/aws/aws-sdk-go-v2/service/docdb v1.49.5
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.58.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.305.3
	github.com/aws/aws-sdk-go-v2/service/ecr v1.58.4
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.39.6
	github.com/aws/aws-sdk-go-v2/service/ecs v1.82.4
	github.com/aws/aws-sdk-go-v2/service/efs v1.42.1
	github.com/aws/aws-sdk-go-v2/service/eks v1.84.6
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.54.3
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.35.4
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.34.6
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.55.4
	github.com/aws/aws-sdk-go-v2/service/elasticsearchservice v1.42.4
	github.com/aws/aws-sdk-go-v2/service/emr v1.61.1
	github.com/aws/aws-sdk-go-v2/service/firehose v1.43.2
	github.com/aws/aws-sdk-go-v2/service/glue v1.143.1
	github.com/aws/aws-sdk-go-v2/service/iam v1.54.4
	github.com/aws/aws-sdk-go-v2/service/identitystore v1.37.7
	github.com/aws/aws-sdk-go-v2/service/iot v1.75.4
	github.com/aws/aws-sdk-go-v2/service/kafka v1.52.6
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.44.2
	github.com/aws/aws-sdk-go-v2/service/kms v1.53.4
	github.com/aws/aws-sdk-go-v2/service/lambda v1.92.3
	github.com/aws/aws-sdk-go-v2/service/medialive v1.98.3
	github.com/aws/aws-sdk-go-v2/service/mediapackage v1.40.1
	github.com/aws/aws-sdk-go-v2/service/mediastore v1.30.3
	github.com/aws/aws-sdk-go-v2/service/memorydb v1.34.6
	github.com/aws/aws-sdk-go-v2/service/mq v1.35.2
	github.com/aws/aws-sdk-go-v2/service/neptune v1.45.4
	github.com/aws/aws-sdk-go-v2/service/opsworks v1.31.0
	github.com/aws/aws-sdk-go-v2/service/organizations v1.51.10
	github.com/aws/aws-sdk-go-v2/service/qldb v1.32.2
	github.com/aws/aws-sdk-go-v2/service/rds v1.119.2
	github.com/aws/aws-sdk-go-v2/service/redshift v1.63.3
	github.com/aws/aws-sdk-go-v2/service/resourcegroups v1.34.2
	github.com/aws/aws-sdk-go-v2/service/route53 v1.63.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.103.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.42.3
	github.com/aws/aws-sdk-go-v2/service/securityhub v1.71.7
	github.com/aws/aws-sdk-go-v2/service/servicecatalog v1.40.4
	github.com/aws/aws-sdk-go-v2/service/ses v1.35.2
	github.com/aws/aws-sdk-go-v2/service/sfn v1.42.2
	github.com/aws/aws-sdk-go-v2/service/sns v1.40.1
	github.com/aws/aws-sdk-go-v2/service/sqs v1.43.2
	github.com/aws/aws-sdk-go-v2/service/ssm v1.69.3
	github.com/aws/aws-sdk-go-v2/service/ssoadmin v1.39.7
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3
	github.com/aws/aws-sdk-go-v2/service/swf v1.34.2
	github.com/aws/aws-sdk-go-v2/service/waf v1.31.5
	github.com/aws/aws-sdk-go-v2/service/wafregional v1.31.4
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.72.4
	github.com/aws/aws-sdk-go-v2/service/workspaces v1.69.1
	github.com/aws/aws-sdk-go-v2/service/xray v1.37.3
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cloudflare/cloudflare-go/v7 v7.4.0
	github.com/cloudfoundry/jibber_jabber v0.0.0-20151120183258-bcc4c8345a21 // indirect
	github.com/ddelnano/terraform-provider-mikrotik/client v0.0.0-20250110092516-5bc3b68c6245
	github.com/ddelnano/terraform-provider-xenorchestra/client v0.18.0-alpha1
	github.com/denverdino/aliyungo v0.0.0-20230411124812-ab98a9173ace
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/digitalocean/godo v1.194.1
	github.com/grafana/grafana-api-golang-client v0.27.0
	github.com/hashicorp/go-azure-helpers v0.79.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-plugin v1.8.0
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/heimweh/go-pagerduty v0.0.0-20250801140645-0b96cfc9bf17
	github.com/heroku/heroku-go/v6 v6.1.0
	github.com/hokaccha/go-prettyjson v0.0.0-20211117102719-0474bc63780f // indirect
	github.com/honeycombio/terraform-provider-honeycombio v0.50.0
	github.com/iancoleman/strcase v0.3.0
	github.com/ionos-cloud/sdk-go-dbaas-mongo v1.4.1
	github.com/ionos-cloud/sdk-go-dbaas-postgres v1.1.4
	github.com/ionos-cloud/sdk-go/v6 v6.3.7
	github.com/jmespath/go-jmespath v0.4.0
	github.com/labd/commercetools-go-sdk v1.9.0
	github.com/microsoft/azure-devops-go-api/azuredevops v1.0.0-b5
	github.com/nicksnyder/go-i18n v1.10.3 // indirect
	github.com/okta/okta-sdk-golang/v6 v6.1.6
	github.com/opsgenie/opsgenie-go-sdk-v2 v1.2.23
	github.com/packethost/packngo v0.31.0
	github.com/pkg/errors v0.9.1
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/tencentyun/cos-go-sdk-v5 v0.7.74
	github.com/yandex-cloud/go-genproto v0.86.0
	github.com/yandex-cloud/go-sdk/v2 v2.118.0
	github.com/zclconf/go-cty v1.18.1
	golang.org/x/oauth2 v0.36.0
	golang.org/x/text v0.38.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.283.0
	google.golang.org/genproto v0.0.0-20260526163538-3dc84a4a5aaa
	k8s.io/apimachinery v0.36.1
	k8s.io/client-go v0.36.1
)

require (
	github.com/IBM-Cloud/container-services-go-sdk v0.0.0-20250409011111-61af13302654
	github.com/IBM/go-sdk-core v1.1.0
	github.com/mackerelio/mackerel-client-go v0.42.0
	github.com/okta/terraform-provider-okta v0.0.0-20260615042503-01f088f45abc
)

require (
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/aws/smithy-go v1.27.2
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
	github.com/go-openapi/strfmt v0.26.3 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
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
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
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
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/OctopusDeploy/go-octodiff v1.0.0 // indirect
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/PuerkitoBio/rehttp v1.4.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/chai2010/gettext-go v1.0.2 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/dnaeon/go-vcr v1.2.0 // indirect
	github.com/dylibso/observe-sdk/go v0.0.0-20240819160327-2d926c5d788a // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/extism/go-sdk v1.7.1 // indirect
	github.com/fluxcd/cli-utils v1.2.0 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
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
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-7 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20240805132620-81f5be970eca // indirect
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
	github.com/lestrrat-go/httprc/v3 v3.0.3 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lestrrat-go/jwx/v3 v3.0.13 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/lib/pq v1.12.3 // indirect
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
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/robertkrimen/otto v0.5.1 // indirect
	github.com/rubenv/sql-migrate v1.8.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.65.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.19.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	k8s.io/apiextensions-apiserver v0.36.0 // indirect
	k8s.io/apiserver v0.36.0 // indirect
	k8s.io/component-base v0.36.0 // indirect
	k8s.io/kubectl v0.36.0 // indirect
	oras.land/oras-go/v2 v2.6.0 // indirect
	sigs.k8s.io/controller-runtime v0.24.0 // indirect
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
	cel.dev/expr v0.25.2 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v1.0.0 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.56.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.56.0 // indirect
	github.com/Myra-Security-GmbH/signature v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/auth0/go-auth0/v2 v2.12.0
	github.com/avast/retry-go v3.0.0+incompatible // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
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
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag v0.26.0 // indirect
	github.com/go-playground/validator/v10 v10.30.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-github/v84 v84.0.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.16 // indirect
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
	google.golang.org/genproto/googleapis/api v0.0.0-20260523011958-0a33c5d7ca68 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260523011958-0a33c5d7ca68 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/validator.v2 v2.0.1 // indirect
	k8s.io/kube-openapi v0.0.0-20260501160325-927ab1f70cd6 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.0 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/analysisservices/armanalysisservices v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6 v6.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8 v8.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance/v2 v2.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry/v3 v3.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3 v3.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datafactory/armdatafactory/v10 v10.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2 v2.0.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mariadb/armmariadb v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9 v9.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/purview/armpurview v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v3 v3.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armlocks v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v3 v3.0.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/security/armsecurity v0.15.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4 v4.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/synapse/armsynapse v0.8.0
	github.com/IBM/continuous-delivery-go-sdk/v2 v2.0.12
	github.com/IBM/go-sdk-core/v3 v3.3.1
	github.com/IBM/go-sdk-core/v4 v4.10.0
	github.com/IBM/sarama v1.50.3
	github.com/aws/aws-msk-iam-sasl-signer-go v1.0.4
	github.com/aws/aws-sdk-go-v2/service/apigatewayv2 v1.35.6
	github.com/aws/aws-sdk-go-v2/service/appconfig v1.44.6
	github.com/aws/aws-sdk-go-v2/service/appintegrations v1.38.2
	github.com/aws/aws-sdk-go-v2/service/appmesh v1.36.4
	github.com/aws/aws-sdk-go-v2/service/apprunner v1.40.6
	github.com/aws/aws-sdk-go-v2/service/appstream v1.60.5
	github.com/aws/aws-sdk-go-v2/service/athena v1.58.4
	github.com/aws/aws-sdk-go-v2/service/backup v1.57.6
	github.com/aws/aws-sdk-go-v2/service/bedrock v1.63.4
	github.com/aws/aws-sdk-go-v2/service/bedrockagent v1.54.6
	github.com/aws/aws-sdk-go-v2/service/chatbot v1.15.6
	github.com/aws/aws-sdk-go-v2/service/chimesdkvoice v1.29.2
	github.com/aws/aws-sdk-go-v2/service/comprehend v1.41.6
	github.com/aws/aws-sdk-go-v2/service/connect v1.176.2
	github.com/aws/aws-sdk-go-v2/service/customerprofiles v1.62.5
	github.com/aws/aws-sdk-go-v2/service/detective v1.39.5
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.39.4
	github.com/aws/aws-sdk-go-v2/service/globalaccelerator v1.36.6
	github.com/aws/aws-sdk-go-v2/service/guardduty v1.79.3
	github.com/aws/aws-sdk-go-v2/service/ivs v1.52.1
	github.com/aws/aws-sdk-go-v2/service/ivschat v1.22.7
	github.com/aws/aws-sdk-go-v2/service/kendra v1.61.3
	github.com/aws/aws-sdk-go-v2/service/lakeformation v1.48.4
	github.com/aws/aws-sdk-go-v2/service/lexmodelbuildingservice v1.36.5
	github.com/aws/aws-sdk-go-v2/service/lexmodelsv2 v1.62.4
	github.com/aws/aws-sdk-go-v2/service/mediaconvert v1.93.1
	github.com/aws/aws-sdk-go-v2/service/mediapackagev2 v1.39.6
	github.com/aws/aws-sdk-go-v2/service/mwaa v1.41.5
	github.com/aws/aws-sdk-go-v2/service/networkmanager v1.42.6
	github.com/aws/aws-sdk-go-v2/service/notifications v1.8.6
	github.com/aws/aws-sdk-go-v2/service/notificationscontacts v1.6.6
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.70.8
	github.com/aws/aws-sdk-go-v2/service/opensearchserverless v1.32.1
	github.com/aws/aws-sdk-go-v2/service/pinpoint v1.40.3
	github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2 v1.29.7
	github.com/aws/aws-sdk-go-v2/service/pipes v1.24.6
	github.com/aws/aws-sdk-go-v2/service/quicksight v1.114.1
	github.com/aws/aws-sdk-go-v2/service/redshiftserverless v1.35.8
	github.com/aws/aws-sdk-go-v2/service/rekognition v1.52.3
	github.com/aws/aws-sdk-go-v2/service/route53resolver v1.45.4
	github.com/aws/aws-sdk-go-v2/service/s3control v1.71.5
	github.com/aws/aws-sdk-go-v2/service/s3tables v1.16.6
	github.com/aws/aws-sdk-go-v2/service/sagemaker v1.253.1
	github.com/aws/aws-sdk-go-v2/service/scheduler v1.18.7
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.62.4
	github.com/aws/aws-sdk-go-v2/service/transcribe v1.56.3
	github.com/aws/aws-sdk-go-v2/service/vpclattice v1.22.2
	github.com/fastly/go-fastly/v15 v15.0.2
	github.com/gofrs/uuid/v3 v3.1.2
	github.com/google/go-github/v88 v88.0.0
	github.com/gophercloud/gophercloud/v2 v2.12.0
	github.com/hashicorp/go-azure-sdk/sdk v0.20260603.1074745
	github.com/hashicorp/vault/api v1.23.0
	github.com/ionos-cloud/sdk-go-cert-manager v1.3.0
	github.com/ionos-cloud/sdk-go-container-registry v1.3.1
	github.com/ionos-cloud/sdk-go-dataplatform v1.1.1
	github.com/ionos-cloud/sdk-go-dns v1.4.0
	github.com/ionos-cloud/sdk-go-logging v1.3.0
	github.com/keycloak/terraform-provider-keycloak v0.0.0-20260526162604-f2be9656a483
	github.com/launchdarkly/api-client-go/v22 v22.0.0
	github.com/linode/linodego/v2 v2.1.0
	github.com/logzio/logzio_terraform_client v1.30.2
	github.com/newrelic/newrelic-client-go/v2 v2.87.1
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/as v1.3.115
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs v1.3.115
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdb v1.3.113
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn v1.3.90
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs v1.3.115
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb v1.3.113
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common v1.3.115
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm v1.3.113
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod v1.3.78
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/es v1.3.104
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/gaap v1.3.34
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/mongodb v1.3.113
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/pts v1.3.29
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/redis v1.3.110
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/scf v1.3.101
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ses v1.3.113
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl v1.3.105
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tat v1.3.107
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tcaplusdb v1.3.105
	github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc v1.3.115
	github.com/vultr/govultr/v3 v3.31.2
	gitlab.com/gitlab-org/api/client-go/v2 v2.36.3
	helm.sh/helm/v4 v4.2.0
	k8s.io/cli-runtime v0.36.1
	software.sslmate.com/src/go-pkcs12 v0.7.2
)

replace gopkg.in/ns1/ns1-go.v2 => github.com/ns1/ns1-go/v2 v2.17.2

replace github.com/tencentcloud/tencentcloud-sdk-go => github.com/tencentcloud/tencentcloud-sdk-go v1.3.115

// Redirect stale transitive imports from abandoned dgrijalva/jwt-go to a compatible maintained fork.
replace github.com/dgrijalva/jwt-go => github.com/form3tech-oss/jwt-go v3.2.5+incompatible
