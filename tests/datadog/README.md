<!-- SPDX-License-Identifier: Apache-2.0 -->

# Testing the Datadog provider

The CLI script creates Datadog resources with Terraform CLI, imports them with
Terraformer, stores generated resources in `generated/`, and verifies the result.

_Note_: The script creates and destroys real resources. Never run this on a
production Datadog organization.

## Requirements

- Terraform CLI 1.9 through 1.14
- Datadog API and application keys for a non-production organization

## Script Usage

Run the script from the project root:

```bash
go run ./tests/datadog/
```

The script should finish without exiting early and should produce no Terraform
plan diff for the generated resources.

## Configuration

| Configuration option | Description |
| --- | --- |
| `DD_TEST_CLIENT_API_KEY` | Datadog API key. |
| `DD_TEST_CLIENT_APP_KEY` | Datadog application key. |
| `DATADOG_HOST` | Datadog API URL. Use `https://api.datadoghq.eu/` for EU sites. Default: `https://api.datadoghq.com/`. |
| `DATADOG_TERRAFORM_TARGET` | Colon-separated resource addresses to target, such as `datadog_dashboard.free_dashboard_example:datadog_monitor.monitor_example`. |
| `LOG_CMD_OUTPUT` | Print Terraform command output to stderr/stdout. Default: `false`. |

## Frequently Asked Questions

```text
Message: Error while importing resources. Error: fork/exec : no such file or directory
```

This means Terraformer could not locate the Datadog provider executable. Run
`terraform init` for the test resources first, or point `TF_DATA_DIR` at a
Terraform 1.x plugin cache containing the Datadog provider.
