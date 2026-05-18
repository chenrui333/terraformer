### Use with Helm

Terraformer imports Helm releases through the HashiCorp `helm` provider.

Supported resources:

* `release`

Example:

```sh
terraformer import helm --resources=release
```

Helm provider authentication follows the standard Helm and Kubernetes client
configuration. Terraformer does not write kubeconfig contents or Kubernetes
credentials into generated configuration; use the current kubeconfig context,
`HELM_*` environment variables, or provider configuration supported by the
HashiCorp `helm` provider.

Release discovery uses Helm SDK action APIs and broad imports search all
namespaces that the current Kubernetes credentials can list. To import a
specific release or avoid broad namespace listing, use the provider-compatible
`namespace/name` import ID:

```sh
terraformer import helm --resources=release --filter=release=default/nginx
```

Multiple exact IDs can be supplied with the normal Terraformer filter value
separator:

```sh
terraformer import helm --resources=release --filter=release=default/nginx:kube-system/metrics-server
```

Broad discovery imports only the latest deployed revision for each release.
Historical or superseded revisions, uninstalled or uninstalling releases,
pending install/upgrade/rollback releases, failed releases, and unknown states
are skipped. Failed releases are not imported because the provider refresh path
is not treated as safe for this lane.

Generated resources seed only fields that can be recovered safely from Helm
release metadata:

* `name`
* `namespace`
* `chart`, when the chart name is present in release metadata
* `version`, when the chart version is present in release metadata
* `description`, when it is a simple non-sensitive status description

Terraformer does not synthesize `repository` because Helm release metadata does
not reliably preserve the authored repository URL or OCI source. Terraformer
also does not export authored `values`, `set`, `set_list`, `set_sensitive`, or
`set_wo` blocks, rendered manifests, Kubernetes Secret data, Helm repository
credentials, OCI credentials, kubeconfig contents, or Kubernetes credentials.
If a release requires these values for future applies, add them manually after
reviewing the original chart and deployment inputs.
