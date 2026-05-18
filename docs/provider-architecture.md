# Terraformer Provider Architecture

## Purpose

This document captures Terraformer provider architecture and lifecycle patterns
for adding and maintaining providers. It is a repo-level guide for the common
wiring, generator, importability, and close-out decisions that provider work
should make explicit.

Use it with the provider-specific docs in `docs/<name>.md` and the unsupported
resource metadata convention in
[unsupported-resources.md](unsupported-resources.md). It does not replace
provider-specific investigation or tests.

## Provider lifecycle

1. Provider planning / tracking issue: define the provider, service families,
   resource buckets, authentication expectations, and known importability risks.
2. Provider skeleton: add the provider directory, command, provider-source
   mapping, registration, docs shell, and wiring tests together.
3. First importable resource: land a small resource or service that proves
   discovery, import ID seeding, refresh, generated HCL, filters, and tests.
4. Feature-gap lanes: split broad providers into focused lanes by service family
   or import shape. Kafka topics and ACLs, Datadog resource families, and
   Cloudflare account/zone/storage lanes are examples of lane-sized work.
5. Unsupported/deferred metadata: record evidence-backed resources that should
   not be imported broadly yet in provider-local `unsupported_resources.json`.
6. Close-out audit: compare code registration, docs, unsupported metadata, issue
   buckets, and stale or superseded PR branches before claiming a lane or
   tracking issue is complete.
7. Long-term maintenance: keep provider docs, provider-source mappings, tests,
   unsupported metadata, and Terraform provider version expectations aligned as
   upstream provider behavior changes.

## New provider wiring checklist

- Provider directory under `providers/<name>`.
- Provider command under `cmd/provider_cmd_<name>.go`.
- Command registration in `cmd/root.go`.
- Provider source mapping in `terraformutils/provider_source.go`.
- Provider source tests in `terraformutils/provider_source_test.go`.
- Provider and service registration tests.
- Docs page under `docs/<name>.md`.
- README provider list update when the provider should appear in the supported
  provider list.
- Optional `unsupported_resources.json` only after evidence-backed unsupported
  or deferred resources exist.

## Generator and service architecture

Service keys in `GetSupportedService()` map CLI `--resources=` values to
Terraform resource generators. For example, Kafka registers `topics` and `acls`
as service keys, while larger providers such as Cloudflare and Datadog group
many Terraform resources behind service-family generators.

Generators should discover remote resources with the provider or domain API.
They should then seed Terraform-provider-compatible import IDs and any required
refresh fields into Terraformer resources. The generated HCL and state must be
refreshable by the upstream Terraform provider.

Apply filters at the service scope before broad, expensive, or
permission-sensitive reads when the API supports scoped reads. Filters should
select or narrow the discovered resources; they should not rewrite global IDs or
normalize provider-facing import IDs destructively.

## Importability decision model

- Full-list import: use when safe list and read APIs exist and Terraformer can
  reconstruct refreshable configuration for every discovered resource.
- Filter-only import: use when no safe broad list API exists, but explicit ID or
  name import is useful, provider-supported, and refreshable.
- Singleton import: use when the provider has a stable singleton ID and the
  refresh context is known. Gate settings resources on explicit user-owned
  configuration, not on effective platform defaults alone.
- Deferred/unsupported: use when a resource is not safely discoverable, has no
  usable import path, represents runtime data or lifecycle actions, is
  high-cardinality row/item/event data, is provider-managed, requires
  secret/write-only fields, or has an unreconstructable body, source, or config.

## Import ID and naming rules

- Preserve upstream Terraform provider import IDs exactly.
- Do not destructively normalize composite, tuple, or delimiter-sensitive IDs.
- Generate Terraform resource names from the full resource identity.
- Add short hashes when name collisions are plausible.
- Skip or defer values that cannot be represented by upstream import syntax.

Kafka ACLs are a useful example: their pipe-delimited tuple IDs must match the
Mongey/kafka importer, and tuple values containing literal `|` characters are
deferred because the importer has no escaping form.

## Sensitive data rules

Terraformer must not export or synthesize these values:

- Passwords.
- Private keys.
- API keys or app keys.
- OAuth tokens.
- Session tokens.
- Kubeconfig contents.
- SCRAM or SASL secrets.
- Repository credentials.
- Rendered secret data.

Use environment variables, profiles, provider configuration, or manual follow-up
instead. Refresh-time provider configuration may carry secrets when the provider
requires them, but generated provider data and generated resources must remain
secret-free.

## Data shape and refresh stability

- Seed fields that the Terraform provider refresh path needs.
- Preserve empty maps, lists, and strings when the provider schema requires them.
- Do not claim platform defaults as Terraform-owned state.
- Gate singleton and settings resources on explicit configured or non-default
  state, not just effective API values.
- Separate durable configuration metadata from runtime, high-cardinality row,
  item, and event data.
- Reject discovered source variants that the Terraform provider cannot validate
  or refresh.

## Unsupported resources metadata

Use provider-local `unsupported_resources.json` files when investigation finds
evidence-backed resources that Terraformer should not import broadly. Keep the
file next to the provider implementation, such as
`providers/<name>/unsupported_resources.json`.

Do not add speculative entries or use metadata as a backlog for missing
implementation work. The schema, allowed status values, evidence expectations,
and validation command are documented in
[unsupported-resources.md](unsupported-resources.md).

## Close-out audit checklist

Before closing a provider tracking issue or declaring a large-provider lane
complete, compare:

- Provider registration.
- Docs supported-resource list.
- Provider-local `unsupported_resources.json`.
- Issue tracker buckets.
- Stale or superseded PR branches.

For broad-provider close-outs, include counts for upstream provider resources,
Terraformer-supported resources, evidence-backed unsupported or deferred
resources, and still-unclassified resources. Separate unclassified resources
already covered by open PRs from truly remaining work, and reduce final
follow-up lanes to exact resource names whenever possible.

A lane can close while the broader tracking issue remains open. A large-provider
issue is closeable only when remaining resources are supported, evidence-backed
deferred or unsupported, or assigned to focused follow-up lanes.
Close the tracking issue as completed only after the final audit confirms docs
and provider registration are aligned, unsupported/deferred metadata validates,
previously named remaining resources are accounted for, the unclassified count
is zero, and validation has passed. Post any close-out summary after that final
validation, not merely after the last expected lane merges.

If a clean audit finds no metadata or documentation corrections, leave the
worktree unchanged instead of creating a docs-only record of the no-op.

## Testing expectations

- Provider source mapping tests.
- Command registration tests.
- Provider and service registration tests.
- Generator unit tests.
- Import ID tests.
- Filter tests.
- Pagination tests.
- Empty, not-found, and access-denied behavior tests.
- Unsupported metadata validation when metadata changes.
