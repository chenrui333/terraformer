# AWS Provider Gap Inventory

Use the AWS gap inventory tool to compare Terraformer AWS resource coverage with
the Terraform AWS provider schema and docs/aws.md.

The tool is read-only. It does not call AWS APIs and does not change Terraformer
import behavior.

## Generate Provider Schema Input

Create a temporary Terraform configuration that only installs the AWS provider:

~~~hcl
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}
~~~

Then generate the provider schema JSON:

~~~bash
terraform init
terraform providers schema -json > aws-provider-schema.json
~~~

## Run The Inventory

From the repository root:

~~~bash
go run ./tools/aws-gap-inventory \
  -provider-schema aws-provider-schema.json \
  -docs docs/aws.md \
  -aws-dir providers/aws \
  -skip-list providers/aws/unsupported_resources.json \
  -format markdown
~~~

Use -format json when another script needs structured output.

If -provider-schema is omitted, the tool still audits docs/aws.md against
resource types detected in providers/aws/*.go, but it cannot report Terraform
AWS provider gaps.

## Skip-List Format

Resources that cannot be imported safely should be recorded in
providers/aws/unsupported_resources.json instead of being added as misleading
Terraformer support.

~~~json
{
  "version": 1,
  "resources": [
    {
      "resource": "aws_example_resource",
      "service_family": "example",
      "reason": "AWS does not expose a list API with enough parent context for safe discovery.",
      "evidence": "Checked Terraform AWS provider schema and AWS service API reference.",
      "status": "unsafe-discovery",
      "references": [
        "https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/example_resource"
      ]
    }
  ]
}
~~~

Supported statuses:

- needs-research
- not-importable
- unsupported
- unsafe-discovery
- deferred

The provider schema lists resource types, not importer contracts. Before adding
an AWS resource importer, still verify the Terraform AWS provider read/import ID
shape, AWS list and describe APIs, pagination, region or global behavior,
generated address uniqueness, filter behavior, and unsupported or deleted states.
