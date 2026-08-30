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
Terraformer support. The canonical status vocabulary and definitions are in
[Unsupported Resource Metadata](unsupported-resources.md#status-values); this
AWS workflow does not define a separate status list.

~~~json
{
  "version": 1,
  "resources": [
    {
      "resource": "aws_example_resource",
      "service_family": "example",
      "reason": "Importability needs a dedicated design for parent-scoped discovery.",
      "evidence": "The AWS list API omits the parent identifier required by the Terraform provider importer.",
      "status": "deferred",
      "references": [
        "https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/example_resource",
        "https://github.com/chenrui333/terraformer/issues/338"
      ]
    }
  ]
}
~~~

Missing Terraformer support alone does not justify an unsupported-resource
record. Every record needs concrete evidence of the import limitation.
`deferred` is appropriate only after investigation identifies specific design
work; `unsupported` or a more specific status requires evidence of the actual
unsafe or non-viable behavior.

The provider schema lists resource types, not importer contracts. Before adding
an AWS resource importer or metadata record, verify:

- Terraform AWS provider importer support.
- The exact import ID and provider Read behavior.
- AWS list and read APIs, including pagination.
- Region or global ownership.
- Generated address uniqueness.
- Filter behavior.
- Deleted and otherwise unsupported states.
