# Terraformer Provider Maintenance

Use this skill for provider-focused work in `chenrui333/terraformer`: feature gap audits, new resource importers, provider-specific bug fixes, docs updates, review iteration, and scoped cleanup.

For repo-level provider architecture and lifecycle guidance, see
[docs/provider-architecture.md](docs/provider-architecture.md).

## Default Operating Mode

1. Confirm repo scope, branch, remotes, and dirty state before editing.
2. Fetch latest `origin/main` and create a fresh sibling worktree from `origin/main` unless the user explicitly says to work in-place.
3. Keep each PR narrow: one provider, one feature family, one bug class, or one docs-only change.
4. Use task-specific branch names and Conventional Commit subjects.
5. Sign commits with `git commit -s`.
6. Use `gh -R chenrui333/terraformer` when creating, reading, or updating GitHub issues and PRs.

## Provider Expansion Workflow

1. Inventory current Terraformer coverage:
   - `docs/<provider>.md`
   - `providers/<provider>/*.go`
   - provider resource maps and service registration
   - `cmd/provider_cmd_<provider>.go`
   - shared provider registry, README, and provider-source tests when the supported-provider list may change
2. Compare Terraformer coverage with the upstream Terraform provider resources, importer behavior, and read paths.
3. Classify candidate resources by import shape before editing:
   - account, zone, project, cluster, or region scoped resources
   - parent/child resources that require seeded parent keys
   - optional add-ons that may be absent, unsupported, or disabled per parent
   - manifest/native resources where generated state must refresh cleanly
   - resources that require beta APIs, custom API versions, or raw HTTP fallbacks
4. For each candidate resource, verify:
   - Terraform provider schema exists at the documented minimum provider version, or the version floor is updated.
   - The Terraform provider read path can refresh from seeded state.
   - The import ID and provider read ID are both understood.
   - Required parent attributes, integration keys, project/environment keys, or config maps are available.
   - The API exposes enough list/read data to reconstruct usable HCL.
   - Pagination, API version headers, beta endpoints, and raw HTTP fallbacks are handled correctly.
   - Generated Terraform addresses remain unique when display names, joined name segments, or child names collide.
5. Prefer high-confidence additions. If a resource cannot be reconstructed accurately, document the gap or skip it rather than emitting misleading config. Docs-only corrections are valid when docs drift from already-supported behavior.

## New External Provider Skeletons

When adding a new non-HashiCorp or external provider, land the whole Terraformer surface together:

- CLI command wiring, provider registration, service registration, provider-source mapping, docs, README/provider list when applicable, and tests for command/service/source wiring.
- Use service or provider API clients for discovery, then seed provider-compatible state for Terraform provider refresh. Do not rely on Terraform provider refresh/import as the inventory discovery mechanism.
- Keep generated provider HCL secret-free. Separate refresh-time provider config from generated provider data: `GetConfig` may need auth fields that the provider requires to refresh, but `GetProviderData` must omit passwords, private keys, access keys, session tokens, OAuth tokens, and similar secret material.
- Prefer environment variables, profiles, or existing provider config paths for authentication instead of synthesizing credentials into generated configuration.
- Apply typed ID filters before broad, expensive, or permission-sensitive list/describe calls when the upstream API supports scoped reads.
- Skip system, internal, default, or provider-managed resources by default unless they are explicitly filtered and provider read confirms they are safely user-owned.
- Partial import is acceptable only when required identity and shape fields are still preserved through refresh/import fallback and the unread fields are optional. If provider refresh cannot recover required fields, defer or mark the resource unsupported.
- Treat cloud-managed variants and provider quirks as evidence-backed filters with tests; skip read-only or unsupported config entries that the Terraform provider intentionally cannot manage.

## Provider Read/Reconstruction Safety

A Terraformer resource is safe to add only when both conditions are true:

1. The Terraform provider has a usable import/read path.
2. The upstream service API returns enough data for the provider read/refresh path to reconstruct every required Terraform configuration field.

Provider import support alone is not sufficient. If provider read state cannot recover required fields, mark the resource unsupported with concrete evidence instead of emitting partial config.

Mark a resource unsupported when:

- the provider requires a template, body, source URI, phrase list, schema, or policy field that the service API does not return;
- the provider read state omits required fields after import;
- the resource requires credentials, private keys, tokens, passwords, webhook secrets, OIDC secrets, or other write-only values;
- the resource represents an operation rather than stable inventory;
- the resource represents an acceptance or handshake lifecycle that cannot be inferred safely from discovery.

## Provider Gap Close-Out And Data Shape Checks

Before claiming a provider gap issue is complete, compare all four sources of truth: Terraformer provider registration, provider docs, provider-local unsupported metadata, and the issue's resource buckets. Also check whether open or stale PR branches were superseded by later merged lanes before continuing work.

Close-out audits are different from feature lanes. They should make the remaining work explicit, not restart parity chasing.

Rules:

- Report lane closure separately from tracking-issue closure. A lane can be complete while the broader issue remains open.
- For large provider schemas, practical close-out means every reviewed candidate is supported, evidence-backed deferred/unsupported, or assigned to a named focused follow-up lane.
- Do not treat literal Terraform provider parity as the goal when resources are request-style, runtime/media output, high-cardinality content, provider-managed, source/body-heavy, or secret-required.
- For settings and singleton resources, distinguish durable user-owned configuration from effective API values and platform defaults before moving a resource from deferred metadata to supported import.
- Use close-out audits to reduce repeated search work: update unsupported metadata when evidence is clear, and group remaining importable resources into focused next lanes.
- If the audit finds no durable metadata or guidance changes, report that directly instead of creating a docs-only PR.

When evaluating product, platform, dataset, table, pipeline, API-definition, or deployment-style resources:

- preserve empty-but-meaningful strings, lists, maps, nested blocks, and variant markers when provider refresh needs them for stable HCL;
- seed nested state that provider read only reconciles when it is already present, such as rule or schedule blocks;
- validate source-shape restrictions before appending discovered IDs, and skip or document source variants the Terraform provider cannot validate or refresh;
- import durable configuration metadata, not high-cardinality rows, items, events, or runtime observations;
- prefer unsupported or deferred metadata over exposing a resource that produces invalid, lossy, or destructive follow-up plans.

## Duplicate Ownership Across Related Resources

Before adding a resource, check whether another Terraformer-supported resource already owns the same upstream object or configuration.

Do not import duplicate ownership paths that would generate two Terraform resources for the same remote configuration.

Defer or skip a resource when:

- two services expose the same underlying catalog, database, dataset, object, workspace, or project;
- a child resource is already managed inline by the parent resource;
- a split policy, tag, or setting resource would conflict with fields already emitted on the parent;
- provider read paths would produce ownership drift between equivalent resources.

## Control-Plane Region Routing

Some provider resources are managed through a specific control-plane region or a limited set of supported API regions, regardless of the user's requested import regions.

Before adding support:

- verify whether the service is regional, global, single-region, or uses a small supported-region set;
- map user-requested regions to the effective API regions;
- de-duplicate effective regions before importing;
- isolate fallback-region behavior in a helper;
- document the routing rule in code comments or tests;
- add tests for region mapping and de-duplication.

## Service Grouping Beyond Global And Regional

Do not assume resources fit only into global and regional buckets.

A provider may need distinct import groups such as:

- global resources;
- single-region control-plane resources;
- fixed-region resources;
- limited-supported-region resources;
- regionless resources exposed through a regional endpoint;
- normal regional resources.

Each group should have:

- explicit grouping logic;
- tests;
- output path isolation;
- clear handling of user-supplied regions.

## Import-Time Provider Configuration

Some services require import-time provider configuration beyond the normal provider setup.

Rules:

- Add a narrow framework hook instead of hard-coding one service into generic import flow.
- Forward the hook through provider facades and wrappers.
- Keep the hook optional and no-op for services that do not need it.
- Reconfigure only the affected service or import scope.
- Restore the original provider configuration after success or failure.
- Preserve the existing provider wrapper if restart or reconfiguration fails.
- Add tests for hook forwarding, error propagation, restore behavior, nil-safe cleanup, and restart-failure preservation.

## Endpoint Bootstrap and Restore Safety

When a service requires endpoint bootstrap:

- discover the service endpoint before resource import;
- reconfigure only the affected service/import scope;
- avoid leaking endpoint overrides into unrelated resources;
- restore the previous endpoint and provider state when the import completes or fails;
- treat expected bootstrap not-found, unavailable, disabled, or not-configured errors as skippable only when the resource family is optional;
- surface unexpected errors.

Add tests covering:

- successful endpoint override;
- bootstrap failure;
- restore after failure;
- provider wrapper preservation when restart fails;
- unrelated services continuing to use the original provider state.

## Framework-Level Hooks Before Provider-Specific Hacks

If a service needs behavior that the import framework does not currently support, prefer a small generic interface or hook over ad hoc code in one provider.

Good hook criteria:

- optional;
- narrow;
- backwards compatible;
- easy to test;
- no-op for existing services;
- implemented by provider/service wrappers without changing unrelated providers.

## Multi-Region Import Path Isolation

When a logical resource group is imported through effective or fallback regions, ensure generated output paths cannot collide with other global, regional, or service-specific imports.

Rules:

- If multiple import buckets run in the same command, isolate output paths by service and/or effective region.
- Do not let fallback-region imports overwrite global or regional output.
- Add tests for path-pattern rewriting.
- Add tests for multi-region grouping when a service uses global, fixed-region, supported-region, or fallback semantics.

## Multi-Region and Replication Import IDs

Replication, failover, and multi-region resources may need local-participant-aware import IDs.

Rules:

- Verify whether provider import/read expects the source, destination, local participant, global identifier, or composite ID.
- For regional import runs, choose the ID that corresponds to the local/imported region when provider behavior requires it.
- Skip replication resources when the local participant cannot be identified.
- Add tests for local-region source and local-region destination cases.
- Add tests for stable replication states and skipped transient/deleted states.

## Shared and Cross-Account Ownership Boundaries

Before importing shared, cross-account, cross-project, or cross-tenant resources, verify that Terraformer can identify ownership safely.

Rules:

- Prefer importing resources owned by the current account, project, tenant, workspace, or organization.
- Preserve full ARNs or globally unique identifiers when provider import/read requires them for shared resources.
- Skip or defer shared resources where ownership is ambiguous.
- Add tests for owner-account, shared-resource, and ARN-vs-ID behavior when applicable.

## Association Discovery Scoping

Association resources should be discovered from the narrowest safe parent scope.

Rules:

- Discover associations through their parent service, network, project, group, workspace, or organization where possible.
- Avoid broad association listing if it can return unrelated, shared, or unowned relationships.
- Seed every parent and child identifier required by provider refresh.
- Use composite import ID helpers and tests for association resources.
- Skip associations in transient, deleting, or failed states unless provider refresh is verified safe.

## Split-Resource Ownership

When a provider exposes split resources for policies, settings, associations, or child configuration, avoid duplicating ownership on the parent resource.

Rules:

- If a split resource owns a policy, body, settings field, or association, remove or ignore that field on the parent where necessary.
- The split resource should seed all provider-required parent identifiers.
- Policy and body fields should be heredoc-wrapped and interpolation-escaped when emitted.
- Optional split-resource discovery should not block parent imports when the split resource is absent or not applicable.

## Default and Provider-Managed Child Resources

Do not emit default or provider-managed child resources as standalone Terraform resources unless the Terraform provider explicitly models and imports them as standalone configuration.

Rules:

- Skip default child resources created implicitly by the platform.
- Skip provider-managed child resources that cannot be updated independently.
- Do not import default listener rules, default policies, default groups, or implicit associations unless provider import/read confirms standalone ownership.
- Add unsupported or deferred notes when default resources appear in discovery but are not valid standalone Terraform resources.

## Customer-Owned vs System-Managed Resources

Skip platform-curated, system-managed, or provider-managed resources unless the Terraform provider explicitly models them as user-owned configuration.

Rules:

- Import customer-created resources.
- Skip provider/system-created defaults, curated pools, curated profiles, system queues, owner memberships, implicit resources, and generated defaults when they are not valid standalone Terraform ownership.
- Do not emit resources that users cannot manage independently.
- Add status or ownership predicates and tests.
- Mark the resource unsupported or deferred when broad discovery cannot distinguish customer-owned resources from system-managed ones.

## High-Cardinality Child Resources

High-cardinality resources require explicit caution before broad default discovery.

Rules:

- Do not add broad default discovery for high-cardinality resources unless the provider already has an established safe pattern.
- Prefer explicit filters, scoped selectors, or focused follow-up PRs.
- Document why the resource is deferred if importing it would create excessive output or slow API scans.
- Add pagination and batching tests when support is eventually added.
- Treat customer and business records as data, not infrastructure inventory, unless they are clearly durable configuration and provider read reconstructs every required field.

## Filter-Gated Definition Resources

Large definition, configuration, and body resources should not always be broadly discovered.

Gate broad discovery behind explicit filters when:

- definitions are large;
- child definitions are high-cardinality;
- API scans are expensive;
- generated output is noisy;
- definition ownership is unclear;
- users are likely to want targeted import only.

Rules:

- Support typed name and ID filters for definition resources.
- Prefilter parent resources when child filters include parent IDs.
- Preserve empty strings and meaningful zero values when the provider read path requires them.
- Do not emit child definition resources for parent-only typed filters unless the parent resource intentionally owns that split output.
- Add tests for filtered and unfiltered behavior.

## Parent Prefiltering From Child Filters

When importing child resources, typed child filters can often identify the required parent IDs.

Rules:

- Parse child filter values to extract parent identifiers when the import ID is composite or when a child filter field contains the parent ID.
- Use extracted parent IDs to narrow parent scans.
- If child filter values cannot be parsed safely, fall back to the broader safe scan or skip with a clear reason.
- Keep parent scans open for mixed filters when global or parent attribute filters need broader evaluation.
- Run the parent scan for child discovery without emitting parent resources unless parent resources were selected.
- Add tests for parent prefilter extraction and mixed filter behavior.

## Mixed Typed and Untyped Filter Behavior

Generators that handle multiple resource types must distinguish:

- typed ID filters;
- typed non-ID filters;
- untyped/global filters;
- mixed parent and child filters.

Rules:

- If a typed filter selects one resource type, only emit matching resources for that type.
- If a child typed filter requires parent discovery, run the parent scan but do not emit parent resources unless selected.
- If global non-ID filters apply, keep discovery broad enough to evaluate them correctly.
- Preserve explicitly requested child resources when global filters match the parent but are not applicable to the child state.
- Do not let sibling resource types survive cleanup just because a typed filter is not applicable to them.
- Add tests for parent-only, child-only, sibling, and mixed typed/global filter combinations.

## Connector and Credential Safety

Connector, channel, platform application, webhook, identity, trust, SAML, OIDC, OAuth, and integration resources often contain write-only or sensitive fields.

Before importing connector resources, inspect provider-required schema fields and service API read output.

Do not import if provider configuration requires values that the service API does not return, including:

- passwords, secrets, client secrets, OIDC secrets, OAuth tokens, API keys;
- private keys, certificate private material;
- credential maps, connection authentication blocks;
- platform credentials, webhook secrets, termination credentials.

If credentials are not recoverable, add an unsupported-resource entry with evidence rather than emitting partial configuration.

## Credential-Backed Identity and Trust Providers

Identity, trust, OAuth, OIDC, SAML, webhook, and connector resources need a variant-by-variant credential review.

Rules:

- Support only non-secret-backed variants when provider read can reconstruct all required fields.
- Split credential-backed variants into unsupported or deferred entries with concrete evidence.
- Do not emit partial trust provider or identity provider resources that would fail plan/apply after refresh.

## Query, SQL, Template, and Body Fields

For resources backed by query text, SQL, templates, JSON policies, scripts, contact-flow content, or other opaque bodies:

- Verify the service API returns the original body or a provider-compatible equivalent.
- Verify provider read preserves enough state for future Terraform plans.
- Heredoc-wrap generated body, query, policy, script, or template strings when the project pattern supports it.
- Escape Terraform interpolation markers such as `${...}`.
- Add tests for body preservation and interpolation escaping.

If the API returns only metadata, checksum, version, name, ARN, or ID but not the actual body/source URI required by Terraform, mark the resource unsupported.

## Provider Validation Constraints in Importability Predicates

Importability predicates should enforce provider validation constraints before emitting a resource.

Check:

- valid lifecycle and status states;
- minimum and maximum values enforced by provider schema;
- required nested blocks;
- mutually exclusive attributes;
- provider-required parent identifiers;
- provider-required role, key, location, or configuration fields.

## Status and Importability Predicates

Every resource family should have explicit status/importability predicates when the API exposes lifecycle state.

Rules:

- Import only stable or provider-refresh-safe states.
- Skip creating, updating, deleting, deleted, and failed resources unless provider read is verified safe.
- Add table-driven tests for allowed and skipped states.
- Treat transient states as deferred, not unsupported, unless provider import is fundamentally impossible.

## Singleton and Default Settings

Account, project, workspace, catalog, dataset, and region singleton settings should be imported carefully.

Rules:

- Verify the provider has an import path.
- Emit singletons only when explicitly configured or when provider semantics require owning the default state.
- Skip default service-managed settings unless the provider represents them as intentional configuration.
- Use preserve-ID metadata when provider read paths normalize singleton IDs differently from Terraformer seed IDs.
- When the API omits an ID for a singleton resource, use a stable synthetic ID (e.g. `"ip-allowlist"`) only when the provider read path does not require a real ID.
- Add tests for default-vs-non-default behavior.

## Empty State Preservation

When a resource has required fields that can be empty strings or zero-count lists, Terraformer's flatmap conversion may strip them.

Rules:

- Add field paths to `AllowEmptyValues` to preserve empty-string attributes that would otherwise be stripped. Note: boolean `"false"` is a non-empty string in flatmap and is NOT dropped — AllowEmptyValues is unnecessary for booleans.
- For required empty lists that `AllowEmptyValues` cannot preserve (zero-count lists are dropped before the allow check), use `PostConvertHook` to set the field to an empty slice (see `IntegrationAWSLogCollectionGenerator` pattern).
- Seed required attributes via `NewResource` when provider refresh depends on context not derivable from the import ID alone (e.g. `org_group_id`, `sink_org_id`, `connection_types`).
- For resources where the only required configuration field is `id`, verify whether `AdditionalFields` or `PostConvertHook` can inject it before marking the resource unsupported.
- Test that empty configurations produce valid HCL, not omitted blocks.

## Mutually Exclusive Nested Blocks

Some provider schemas represent alternatives as mutually exclusive nested blocks, wildcard blocks, expressions, or lists.

Rules:

- Preserve the provider-compatible representation returned by the read path.
- Drop conflicting fields when a wildcard, all, or any block is present.
- Add tests for wildcard, all, empty expression, and empty-list behavior.
- Do not emit both a wildcard block and a concrete expression or list if the provider treats them as conflicting.

## Acceptance and Handshake Lifecycle Resources

Do not broadly discover/import resources whose Terraform resource represents accepting, approving, joining, or acknowledging a relationship unless the provider read path can represent that relationship without side effects and without guessing ownership.

Rules:

- Do not call mutating acceptance APIs during import discovery.
- Do not infer that Terraform should own an acceptance just because a relationship exists.
- Prefer unsupported or deferred entries with evidence when acceptance lifecycle ownership is ambiguous.

## Organization/Admin Scoped Optional Loaders

For resources that require organization, admin, delegated-admin, subscription, premium-plan, or centralized-configuration privileges:

- Load base resources first.
- Run admin/organization child loaders as optional loaders after base support is verified.
- Treat expected non-admin, not-enabled, not-subscribed, not-found, and centralized-configuration errors as skippable when the base resource can still be imported safely.
- Surface unexpected permission, API, parsing, and pagination errors.
- Add tests that distinguish expected optional-loader errors from real failures.

## Optional Loaders and Partial Discovery Failures

For large multi-resource generators:

- load base resources first;
- make child and optional discovery resilient;
- run policy, body, association, and child loaders as optional when APIs may return not found, not enabled, not supported, not applicable, or unsupported-region errors;
- skip expected not-found, access-denied, not-enabled, unsupported-region, not-applicable, and default/unconfigured errors for optional resource groups;
- surface unexpected errors;
- do not let one optional child API failure block unrelated imports.

## Pagination Requirements for Child Discovery

Every list/describe API used for broad discovery must be checked for pagination.

Rules:

- Use paginators where available.
- Verify the pagination base per API; do not assume page 1 — some APIs use zero-based page numbers.
- Add pagination tests for child-resource discovery when a previous implementation used a single page.
- Do not assume child-resource lists fit in one response.
- When an API has nested pagination under each parent, test parent and child pagination together.

## Preserve ID After Refresh

Use preserve-ID metadata only when verified provider refresh behavior requires it.

Common cases:

- framework or singleton resources whose provider read path normalizes the ID;
- composite IDs where read state does not preserve the exact import string;
- settings resources with synthetic import IDs.

Always add a test or documented evidence when using preserve-ID behavior.

## Provider Importer ID Verification

Always verify the exact ID shape expected by the provider importer and read path.

Do not assume these are correct for broad discovery without checking provider source and read behavior:

- display names, ARNs, short IDs, composite IDs, human-readable names, provider docs examples.

Rules:

- Add explicit import ID helper functions.
- Add tests for every import ID shape.
- Correct existing import IDs when provider read requires a different identifier.
- Verify whether provider read expects an internal ID even when the API displays a name or ARN.
- Preserve IDs after refresh only when provider behavior requires it and evidence exists.

## Import ID Uniqueness

A provider importer may exist but still be unsafe for broad discovery if the import ID is not unique enough to distinguish discovered resources.

Before adding broad discovery:

- verify import ID uniqueness across account, project, region, namespace, and parent scopes;
- include parent identifiers when needed;
- add collision tests for generated resource names;
- mark unsupported or defer when the provider importer uses an ambiguous identifier.

## Composite Import ID Preservation

When the provider importer requires a composite ID, preserve every semantic component required by provider read.

Rules:

- Do not simplify composite IDs to a single human-readable name if provider read needs both parent and child identifiers.
- Add explicit helper functions for import ID construction.
- Add tests for every composite import ID shape.
- Use stable resource names that include enough parent/child context to avoid sanitized name collisions.
- Treat composite and tuple ID filters as literal import identities; do not pass them through generic parsers that split on delimiters which may also appear inside tuple fields unless the provider format guarantees escaping.
- If a discovered identity contains a delimiter that the upstream importer cannot escape or represent, skip or defer it with evidence instead of emitting a broken import ID.
- If provider refresh normalizes or drops an import ID, use the project's ID preservation metadata only after verifying the behavior.

## Parent/Child Dependency References

For child resources that require a parent service, account, project, organization, or resource to exist:

- Seed parent identifiers in attributes when the provider read path expects them.
- Add dependency references only when generated config would otherwise be ordered incorrectly or fail refresh.
- Keep dependency references narrow and deterministic.
- Add tests for generated parent references if the generator constructs or rewrites them.

## Unsupported Resource Evidence Quality

Unsupported-resource entries should explain the exact blocking condition.

Good evidence includes:

- provider requires field `X`;
- service API `Y` does not return `X`;
- provider has no importer;
- provider read cannot recover field `X` after import;
- resource is mutating/action-style or acceptance lifecycle state;
- resource requires write-only secret material;
- provider import ID is ambiguous for broad discovery;
- resource represents high-cardinality business or customer data rather than durable infrastructure configuration.

Avoid vague entries such as "not supported", "hard to import", "complex", "maybe unsafe".

## No-Importer Unsupported Rule

If the Terraform provider has no importer for a resource, do not add partial Terraformer support.

Rules:

- Mark the resource unsupported or deferred with evidence.
- Include provider version/source context when useful.
- Do not fake support through docs-only entries.
- Do not rely on seeded partial state when the provider has no supported import path.
- Re-check in later provider versions if the resource is high-value.

## Typed Filter Handling

For generators that emit several Terraform resource types, typed filters must be scoped to the current Terraform resource type before calling `terraformutils.ResourceFilter.Filter`.

Rules:

- If there are no typed filters for the service family, load all resource families.
- If there is a typed filter for one resource type, only load and append resources for that type unless an untyped/global filter requires broader discovery.
- Untyped/global filters should keep all relevant resource-family loaders available so post-refresh cleanup can evaluate them.
- Normalize resource service names consistently, including `aws_`-prefixed and non-prefixed forms where existing code supports both.
- Add tests proving typed filters for one resource type do not accidentally include or exclude sibling resource types.

## Resource Name Collision Prevention

When building Terraform resource names from ARNs, slash-delimited IDs, composite IDs, parent/child IDs, or shared-resource identifiers, use length-prefixed parts or another collision-resistant scheme.

Add collision tests, especially where:

- IDs contain slashes, commas, colons, underscores, or hyphens;
- ARNs or globally unique identifiers contain repeated path components;
- parent and child resources share human-readable names;
- generated names combine multiple parent identifiers;
- sanitized names could collapse two distinct remote objects into the same Terraform address.

## AI/ML and Training-Backed Resources

AI/ML resources that represent trained models, classifiers, recognizers, language models, projects, processors, or similar training-backed artifacts need explicit status/importability predicates.

Import only stable/readable states where provider refresh is safe. Defer or mark unsupported when:

- the resource is still training, creating, updating, stopping, or deleting;
- failed resources are not known to refresh safely through the provider;
- training source data, vocabulary/source file fields, or generated content are required but not recoverable;
- runtime/job artifacts would create noisy lifecycle inventory rather than stable infrastructure state.

## Provider-Read Normalization Conflicts

If provider read state normalizes fields in a way that conflicts with generated HCL, add a narrow ignore rule and a regression test.

Only add ignore rules after verifying provider read behavior.

## Large Multi-Service Sweep Strategy

For large lane PRs that span several related services:

- start with an implementation matrix;
- expect each long-tail service to have distinct endpoint, ownership, filter, and import-ID behavior;
- implement resources in coherent groups;
- run narrow tests after each group;
- continue to the next group only when import IDs, provider refresh behavior, generated names, and tests are solid;
- prefer simple customer-owned resources first;
- isolate framework-level changes from resource support where possible;
- add unsupported entries for no-importer and unsafe resources;
- add strong helper tests because service-specific behavior is less likely to be covered by existing patterns;
- defer schema-heavy service families that need focused handling instead of forcing fragile support into a broad sweep.

## Implementation Rules

- Seed state with the ID shape expected by the Terraform provider read path, not necessarily the human import ID.
- Seed required attributes that the provider read path needs before refresh, especially parent keys, integration keys, project/environment keys, and required config maps.
- Preserve scope in both discovery and generated names.
- Page through every collection endpoint. Do not treat page-local counts as collection totals.
- Include stable unique keys or IDs in generated Terraform resource names when display names or joined name segments can collide.
- Route discovery through the provider-compatible control-plane region when it differs from the requested import region, and de-duplicate repeated effective regions.
- Isolate output paths when global, fixed-region, fallback-region, or service-specific import passes run separately in one command.
- Treat ambiguous provider import IDs as unsafe for broad discovery even when an importer exists.
- Treat expected per-resource absences, unsupported optional states, and disabled child features as skips when the Terraform resource cannot represent them.
- For optional loaders, skip only expected unavailable/not-enabled/non-admin errors and let unexpected IAM, API, pagination, or parsing errors fail loudly.
- Avoid generating conflicting ownership models in docs or examples.
- Close response bodies where existing Terraformer patterns require it.
- Use generated clients when they expose the endpoint correctly. Use raw HTTP only for missing endpoints, missing query support, or custom API-version behavior.
- Use a client configured for the API version required by the endpoint.
- For manifest-style or native resources, verify that seeded state can be converted back into provider-compatible HCL before documenting support.
- Keep docs aligned with actual service names, resource groups, provider resource types, and management-mode caveats.

## Session Wrap and Maintainer Handoff

Every implementation lane must end with a concise session wrap that another maintainer can reuse without reading the full transcript:

- resources added;
- resources skipped or marked unsupported;
- import ID shapes verified;
- APIs used;
- reusable helper patterns;
- new skip-list patterns;
- tests added;
- central files touched;
- merge-conflict risk;
- skill update candidates;
- suggested exact skill wording.

Every review lane must end with a review session wrap:

- bugs found;
- root cause;
- generalizable prevention rule;
- missing or added tests;
- suggested exact skill wording.

## Review-Lane Bug Patterns

When reviewing provider importer changes, look first for:

- resources silently omitted by over-broad status filters, typed filters, or optional-loader skip guards;
- optional loaders that swallow generic permission errors instead of only expected unavailable cases;
- service clients that use the requested region even though provider refresh uses a control-plane, fixed, or fallback region;
- multi-region import passes that can write different resource groups into the same output path and overwrite state or provider files;
- generated HCL that loses required provider fields after refresh;
- generated state/HCL mismatches after post-convert fixes;
- dependency references that point at sanitized names that do not exist;
- split resources that duplicate parent ownership of policies, settings, or associations;
- provider import IDs that are not unique enough for broad discovery;
- composite IDs reconstructed from partial ARNs, names, or suffixes instead of preserving all provider-required components;
- per-item API failures that should skip only the stale child resource, not abort or loop the whole collection.

## Validation

Use the narrowest validation that proves the change:

```bash
git diff --check
go test ./providers/<provider>
go test ./cmd ./providers/<provider>
golangci-lint run --new-from-rev="$(git merge-base HEAD origin/main)" ./providers/<provider>
```

Broaden validation when touching shared provider helpers, Go modules, generated files, or command registration:

```bash
go mod tidy
go test ./...
go build -v
```

For docs-only changes, `git diff --check` is usually enough unless links, generated docs, or examples need a targeted check.

---

## Provider Maintenance Checklists

### Scope Setup

- Verify the checkout and GitHub target:
  - `git remote -v`
  - `git status --short --branch`
  - `gh repo view -R chenrui333/terraformer --json nameWithOwner,defaultBranchRef`
- Refresh `origin/main` before branching:
  - `git fetch origin main --prune`
  - `git worktree add ../terraformer-<provider>-<topic> origin/main`
- Keep the existing main checkout untouched if it has unrelated dirty files.

### Feature Gap Audit

- Compare Terraformer support against the upstream Terraform provider:
  - resource names in provider docs or registry
  - provider source schema/import/read code
  - Terraformer docs resource groups
  - Terraformer provider service registration
  - shared provider lists, source mappings, and tests when adding a new provider or command
- Group possible gaps by blast radius:
  - docs-only advertised resource groups
  - simple account-level resources
  - project-scoped resources
  - project/environment/flag-scoped resources
  - cluster, namespace, or native/manifest-style resources
  - parent/child resources with optional child collections
  - resources requiring beta APIs or raw HTTP fallbacks
- Prefer resources with a list endpoint, deterministic read path, and reconstructable config.
- Skip or document resources where the API exposes only derived/computed state or where required config is not recoverable.

### Terraform Provider Compatibility

- Check that the documented minimum provider version includes every advertised Terraform resource type.
- If adding resources requires a provider version bump, update all relevant docs and examples in the same PR.
- Read the Terraform provider importer and read implementation before setting Terraformer state.
- Distinguish:
  - human import ID accepted by `terraform import`
  - provider read ID stored in `d.Id()`
  - parent attributes needed for read calls
- Seed all attributes that the provider read path expects to already exist.
- For maps or nested config, preserve provider-specific key names exactly.
- For parent/child resources, test or inspect both the child import ID and the provider's post-refresh ID.
- For optional resources, confirm the provider can represent the discovered state.
- For manifest-style resources, verify the flatmap or object payload can be converted into provider-compatible HCL.

### API And Pagination

- Prefer generated API clients when they support the endpoint correctly.
- Use raw HTTP only when the generated client lacks the endpoint, query support, or required API version behavior.
- For every list endpoint, identify the pagination contract (next link, offset/limit, cursor, page size, total count semantics).
- Page until the API indicates completion. Be careful with page-local fields named `totalCount`.
- Treat 404 as empty only when the scope genuinely may not support the child collection.
- Close response bodies consistently with the surrounding provider code.
- For beta endpoints, ensure the request sends only the required API version header.

### Generated Names And Ownership

- Generated Terraform addresses must be unique for all resources returned by the API.
- Include stable parent keys and resource IDs when names can collide.
- Do not rely on display names as unique keys.
- Do not rely on joining name segments with `_` when sanitized source values may already contain underscores.
- If generated resources reference other generated resources, build references from the same sanitized label function.
- Avoid examples that import two ownership models for the same remote object.

### Implementation Review

- New service is registered in the provider resource group map.
- CLI resource list includes the new resource group where appropriate.
- Docs list the emitted Terraform resource types.
- Provider-source mapping, shared generator registry, and README-supported provider lists are updated when the provider surface changes.
- Generated resource names are stable and collision-resistant.
- State seeds match the provider read path.
- Required config is preserved after refresh.
- Pagination covers more than one page.
- Optional child lookups skip expected absence without hiding real parent-scope failures.
- API errors are neither swallowed nor over-reported.
- Tests cover risky ID, pagination, naming, or config behavior when practical.

### Validation Templates

Provider-scoped checks:

```bash
git diff --check
go test ./providers/<provider>
go test ./cmd ./providers/<provider>
golangci-lint run --new-from-rev="$(git merge-base HEAD origin/main)" ./providers/<provider>
```

Shared or dependency checks:

```bash
go mod tidy
git diff -- go.mod go.sum
go test ./...
go build -v
```

### PR Loop

- Keep PR body concise: `Summary` and optional `Notes` or `References`.
- Let CI run broad build/lint when the user wants fast review iteration.
- When review comments arrive, reproduce or inspect the exact code path before patching.
- Add follow-up commits on squash-merge PR branches unless a force-push is explicitly needed.
- After merge, refresh `main` before starting the next provider gap.

## Terraformer Framework Gotchas

Durable lessons from provider-gap implementation that prevent repeated review cycles.

### AllowEmptyValues for Required Booleans

Terraformer's flatmap parser drops zero-value attributes unless they appear in `AllowEmptyValues`. When a Terraform resource has required boolean fields that can legitimately be `false`, list them in `AllowEmptyValues` or the generated HCL omits required arguments and `terraform plan` fails.

### AcceptableValues Rewrite for Composite ID Filters

Terraformer's `InitialCleanup` compares `filter.AcceptableValues` against `resource.InstanceState.ID`. When a filter uses a composite format (e.g., `parent_id:child_id`) but the resource state ID is only the child part, rewrite `g.Filter[filterIndex].AcceptableValues` to bare IDs after parsing. See `rum_retention_filter.go` for the canonical pattern.

### Terraformer Strips the id Attribute During HCL Conversion

Terraformer unconditionally ignores `InstanceState.Attributes["id"]` when generating HCL. If a Terraform resource's only Required configuration argument is literally `id`, it cannot produce valid HCL through Terraformer. Mark such resources as unsupported with evidence.

### Metadata-Only Key Imports

API key, application key, and similar secret-bearing resources are importable when the Terraform provider marks the secret field as `Computed+Sensitive` (not `Required`). The provider refresh works without the original secret value; generated HCL contains only metadata (name, dates). Document this behavior in `docs/<provider>.md`. Never export actual key/secret values.

### Unstable API Operations

Some SDK endpoints are gated behind `configuration.SetUnstableOperationEnabled`. When a list/get API requires this, call it on the client configuration before creating the API instance. Document the unstable status in a code comment and test helper.
