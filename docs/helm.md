### Use with Helm

Terraformer has initial Helm provider wiring for the `helm` provider. Helm
release discovery and import support is tracked in issue #489 and is not
implemented yet.

Reserved service key:

* `release`

The `release` service key is reserved for future `helm_release` import support.
This skeleton intentionally does not discover Helm releases or emit
`helm_release` resources.

Follow-up release import work should use Helm SDK discovery and provider
compatible `namespace/name` import IDs.
