### Use with LaunchDarkly

Example:

```
export LAUNCHDARKLY_ACCESS_TOKEN=[LAUNCHDARKLY_ACCESS_TOKEN]
./terraformer import launchdarkly -r environment,featureFlag,segment
```

Use `project` separately when you want LaunchDarkly environments managed as nested
`launchdarkly_project` blocks. Avoid importing `project` and `environment` together,
because that can generate both nested and standalone resources for the same
environments.

List of supported LaunchDarkly resources:

*   `project`
    * `launchdarkly_project`
*   `environment`
    * `launchdarkly_environment`
*   `featureFlag`
    * `launchdarkly_feature_flag`
    * `launchdarkly_feature_flag_environment`
*   `segment`
    * `launchdarkly_segment`
