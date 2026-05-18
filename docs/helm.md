### Use with Helm

Terraformer imports Helm releases through the HashiCorp `helm` provider.

Supported resources:

* `release`

Example:

```sh
terraformer import helm --resources=release
```

Helm provider authentication follows the standard Helm and Kubernetes client
configuration. Terraformer uses the same current kubeconfig path, `KUBECONFIG`,
`KUBE_CONFIG_PATH`, or `KUBE_CONFIG_PATHS` setting for discovery and provider
refresh, and mirrors equivalent `KUBE_*` and `HELM_*` context/server/token
environment variables when needed. When both provider-style `KUBE_*` variables
and Helm/client-go variables are set, the provider-style kubeconfig and context
settings are used for discovery so discovery and refresh target the same
cluster. Provider-style host, token, CA data, client certificate, basic auth,
proxy, and kubeconfig context override environment variables are also applied to
Helm discovery. Terraformer does not write kubeconfig contents or Kubernetes
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

Broad discovery imports a release only when its latest revision is in
`deployed` state; otherwise the release is skipped. Historical or superseded
revisions, uninstalled or uninstalling releases, pending install/upgrade/rollback
releases, failed releases, and unknown states are skipped. Failed releases are
not imported because the provider refresh path is not treated as safe for this
lane.

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
Provider refresh may recover Helm metadata from the live release; Terraformer
strips value-bearing release metadata before writing generated state.
If a release requires these values for future applies, add them manually after
reviewing the original chart and deployment inputs.
