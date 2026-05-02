### Use with LaunchDarkly

Example:

```
export LAUNCHDARKLY_ACCESS_TOKEN=[LAUNCHDARKLY_ACCESS_TOKEN]
./terraformer import launchdarkly -r project,environment,featureFlag,segment
```

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
