// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/providerwrapper"
	"github.com/zclconf/go-cty/cty"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	k8sclient "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // GKE support
)

type KubernetesProvider struct { //nolint
	terraformutils.Provider
	verbose string
}

func (p KubernetesProvider) GetResourceConnections() map[string]map[string][]string {
	return map[string]map[string][]string{}
}

func (p KubernetesProvider) GetProviderData(_ ...string) map[string]interface{} {
	return map[string]interface{}{}
}

func (p *KubernetesProvider) Init(args []string) error {
	p.verbose = ""
	if len(args) > 0 {
		p.verbose = args[0]
	}
	return nil
}

func (p *KubernetesProvider) GetName() string {
	return "kubernetes"
}

func (p *KubernetesProvider) InitService(serviceName string, verbose bool) error {
	p.Service = nil

	service, isSupported := p.GetSupportedService()[serviceName]
	if !isSupported {
		return errors.New("kubernetes: " + serviceName + " not supported resource")
	}
	p.Service = service
	p.Service.SetName(serviceName)
	p.Service.SetVerbose(verbose)
	p.Service.SetProviderName(p.GetName())
	return nil
}

// GetSupportService return map of supported resource for Kubernetes
func (p *KubernetesProvider) GetSupportedService() map[string]terraformutils.ServiceGenerator {
	resources := make(map[string]terraformutils.ServiceGenerator)

	config, _, err := initClientAndConfig()
	if err != nil {
		return resources
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Println(err)
		return resources
	}

	lists, err := kubernetesPreferredResources(dc)
	if err != nil {
		log.Println(err)
		return resources
	}
	clientset, err := k8sclient.NewForConfig(config)
	if err != nil {
		log.Println(err)
		return resources
	}
	provider, err := providerwrapper.NewProviderWrapper("kubernetes", cty.Value{}, p.verbose == "true")
	if err != nil {
		log.Println(err)
		return resources
	}
	resp := provider.GetSchema()
	listableResources := map[kubernetesResourceID]struct{}{}
	envResources := map[kubernetesResourceID]struct{}{}
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}

		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}

			// filter to resources that support list
			if len(resource.Verbs) > 0 && !sets.NewString(resource.Verbs...).Has("list") {
				continue
			}
			listableResources[kubernetesResourceID{group: gv.Group, version: gv.Version, kind: resource.Kind}] = struct{}{}
			if envSupportsResource(resource) {
				envResources[kubernetesResourceID{group: gv.Group, version: gv.Version, kind: resource.Kind}] = struct{}{}
			}

			hasResourceType := func(name string) bool {
				_, exists := resp.ResourceTypes[name]
				return exists
			}
			terraformResourceName, useDynamicClient, ok := selectImportResourceName(clientset, gv.Group, gv.Version, resource, hasResourceType)
			if !ok {
				continue
			}

			addKubernetesResourceService(resources, gv.Group, gv.Version, resource, terraformResourceName, useDynamicClient)
		}
	}
	addDefaultServiceAccountService(resources, clientset, listableResources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	addNodeTaintService(resources, clientset, listableResources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	addConfigMapDataService(resources, clientset, listableResources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	addSecretDataService(resources, clientset, listableResources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	addEnvService(resources, envResources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	addMetadataPatchServices(resources, func(name string) bool {
		_, exists := resp.ResourceTypes[name]
		return exists
	})
	return resources
}

func addKubernetesResourceService(
	resources map[string]terraformutils.ServiceGenerator,
	group string,
	version string,
	resource metav1.APIResource,
	terraformResourceName string,
	useDynamicClient bool,
) {
	resources[kubernetesResourceServiceKey(group, version, resource.Name, terraformResourceName)] = &Kind{
		Group:            group,
		Version:          version,
		Name:             resource.Kind,
		ResourceName:     resource.Name,
		Namespaced:       resource.Namespaced,
		TerraformType:    terraformResourceName,
		UseDynamicClient: useDynamicClient,
	}
}

func kubernetesResourceServiceKey(group, version, resourceName, terraformResourceName string) string {
	if terraformResourceName != manifestTerraformResourceName {
		return resourceName
	}
	if group == "" {
		return version + "/" + resourceName
	}
	return group + "/" + version + "/" + resourceName
}

func addDefaultServiceAccountService(
	resources map[string]terraformutils.ServiceGenerator,
	clientset k8sclient.Interface,
	listableResources map[kubernetesResourceID]struct{},
	hasResourceType func(string) bool,
) {
	if _, ok := listableResources[kubernetesResourceID{version: "v1", kind: "ServiceAccount"}]; !ok {
		return
	}
	if !supportsTypedClientResource(clientset, "", "v1", "ServiceAccount") {
		return
	}

	terraformResourceName, ok := selectTerraformResourceName("", "v1", defaultServiceAccountKind, hasResourceType)
	if !ok {
		return
	}

	resources[defaultServiceAccountServiceName] = &DefaultServiceAccount{
		TerraformType: terraformResourceName,
	}
}

func addNodeTaintService(
	resources map[string]terraformutils.ServiceGenerator,
	clientset k8sclient.Interface,
	listableResources map[kubernetesResourceID]struct{},
	hasResourceType func(string) bool,
) {
	if _, ok := listableResources[kubernetesResourceID{version: "v1", kind: "Node"}]; !ok {
		return
	}
	if !supportsTypedClientResource(clientset, "", "v1", "Node") {
		return
	}
	if !hasResourceType(nodeTaintTerraformType) {
		return
	}

	resources[nodeTaintServiceName] = &NodeTaint{
		TerraformType: nodeTaintTerraformType,
	}
}

func addConfigMapDataService(
	resources map[string]terraformutils.ServiceGenerator,
	clientset k8sclient.Interface,
	listableResources map[kubernetesResourceID]struct{},
	hasResourceType func(string) bool,
) {
	if _, ok := listableResources[kubernetesResourceID{version: "v1", kind: "ConfigMap"}]; !ok {
		return
	}
	if !supportsTypedClientResource(clientset, "", "v1", "ConfigMap") {
		return
	}
	if !hasResourceType(configMapDataTerraformType) {
		return
	}

	resources[configMapDataServiceName] = &ConfigMapData{
		TerraformType: configMapDataTerraformType,
	}
}

func addSecretDataService(
	resources map[string]terraformutils.ServiceGenerator,
	clientset k8sclient.Interface,
	listableResources map[kubernetesResourceID]struct{},
	hasResourceType func(string) bool,
) {
	if _, ok := listableResources[kubernetesResourceID{version: "v1", kind: "Secret"}]; !ok {
		return
	}
	if !supportsTypedClientResource(clientset, "", "v1", "Secret") {
		return
	}
	if !hasResourceType(secretDataTerraformType) {
		return
	}

	resources[secretDataServiceName] = &SecretData{
		TerraformType: secretDataTerraformType,
	}
}

func addEnvService(
	resources map[string]terraformutils.ServiceGenerator,
	envResources map[kubernetesResourceID]struct{},
	hasResourceType func(string) bool,
) {
	if !hasResourceType(envTerraformType) {
		return
	}
	for resourceID := range envResources {
		if envSupportsKind(resourceID.kind) {
			resources[envServiceName] = &Env{
				TerraformType: envTerraformType,
			}
			return
		}
	}
}

func addMetadataPatchServices(
	resources map[string]terraformutils.ServiceGenerator,
	hasResourceType func(string) bool,
) {
	if hasResourceType(labelsTerraformType) {
		resources[labelsServiceName] = &MetadataPatch{
			TerraformType:     labelsTerraformType,
			AttributeName:     "labels",
			AllowEmptyPattern: labelsAllowEmptyPattern,
		}
	}
	if hasResourceType(annotationsTerraformType) {
		resources[annotationsServiceName] = &MetadataPatch{
			TerraformType:     annotationsTerraformType,
			AttributeName:     "annotations",
			AllowEmptyPattern: annotationsAllowEmptyPattern,
		}
	}
}

func (p KubernetesProvider) PostProcessImportResources(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	resourcesByService = removeDefaultServiceAccountDuplicates(resourcesByService)
	resourcesByService = removeConfigMapDataDuplicates(resourcesByService)
	resourcesByService = removeSecretDataDuplicates(resourcesByService)
	resourcesByService = removeEnvDuplicates(resourcesByService)
	resourcesByService = removeMetadataPatchDuplicates(resourcesByService)
	return resourcesByService
}

func removeDefaultServiceAccountDuplicates(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	defaultServiceAccountIDs := map[string]struct{}{}
	for _, resource := range resourcesByService[defaultServiceAccountServiceName] {
		if resource.InstanceState == nil {
			continue
		}
		defaultServiceAccountIDs[resource.InstanceState.ID] = struct{}{}
	}
	if len(defaultServiceAccountIDs) == 0 {
		return resourcesByService
	}

	serviceAccounts, ok := resourcesByService["serviceaccounts"]
	if !ok {
		return resourcesByService
	}
	filtered := serviceAccounts[:0]
	for _, resource := range serviceAccounts {
		if isDefaultServiceAccountDuplicate(resource, defaultServiceAccountIDs) {
			continue
		}
		filtered = append(filtered, resource)
	}
	resourcesByService["serviceaccounts"] = filtered
	return resourcesByService
}

func removeConfigMapDataDuplicates(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	configMaps, ok := resourcesByService["configmaps"]
	if !ok {
		return resourcesByService
	}
	configMapIDs := map[string]struct{}{}
	for _, resource := range configMaps {
		if resource.InstanceState != nil {
			configMapIDs[resource.InstanceState.ID] = struct{}{}
		}
	}
	if len(configMapIDs) == 0 {
		return resourcesByService
	}

	configMapData, ok := resourcesByService[configMapDataServiceName]
	if !ok {
		return resourcesByService
	}
	filtered := configMapData[:0]
	for _, resource := range configMapData {
		if resource.InstanceState == nil {
			filtered = append(filtered, resource)
			continue
		}
		if _, duplicate := configMapIDs[resource.InstanceState.ID]; duplicate {
			continue
		}
		filtered = append(filtered, resource)
	}
	if len(filtered) == 0 {
		delete(resourcesByService, configMapDataServiceName)
		return resourcesByService
	}
	resourcesByService[configMapDataServiceName] = filtered
	return resourcesByService
}

func removeSecretDataDuplicates(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	secrets, ok := resourcesByService["secrets"]
	if !ok {
		return resourcesByService
	}
	secretIDs := map[string]struct{}{}
	for _, resource := range secrets {
		if resource.InstanceState != nil {
			secretIDs[resource.InstanceState.ID] = struct{}{}
		}
	}
	if len(secretIDs) == 0 {
		return resourcesByService
	}

	secretData, ok := resourcesByService[secretDataServiceName]
	if !ok {
		return resourcesByService
	}
	filtered := secretData[:0]
	for _, resource := range secretData {
		if resource.InstanceState == nil {
			filtered = append(filtered, resource)
			continue
		}
		if _, duplicate := secretIDs[resource.InstanceState.ID]; duplicate {
			continue
		}
		filtered = append(filtered, resource)
	}
	if len(filtered) == 0 {
		delete(resourcesByService, secretDataServiceName)
		return resourcesByService
	}
	resourcesByService[secretDataServiceName] = filtered
	return resourcesByService
}

func removeEnvDuplicates(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	env, ok := resourcesByService[envServiceName]
	if !ok {
		return resourcesByService
	}

	targetIDs := map[string]struct{}{}
	for serviceName, resources := range resourcesByService {
		if serviceName == envServiceName {
			continue
		}
		for _, resource := range resources {
			for _, targetID := range envTargetIDs(resource) {
				targetIDs[targetID] = struct{}{}
			}
		}
	}
	if len(targetIDs) == 0 {
		return resourcesByService
	}

	filtered := env[:0]
	for _, resource := range env {
		if resource.InstanceState == nil {
			filtered = append(filtered, resource)
			continue
		}
		if _, duplicate := targetIDs[resource.InstanceState.ID]; duplicate {
			continue
		}
		filtered = append(filtered, resource)
	}
	if len(filtered) == 0 {
		delete(resourcesByService, envServiceName)
		return resourcesByService
	}
	resourcesByService[envServiceName] = filtered
	return resourcesByService
}

func removeMetadataPatchDuplicates(resourcesByService map[string][]terraformutils.Resource) map[string][]terraformutils.Resource {
	targetIDs := map[string]struct{}{}
	targetObjectKeys := map[string]struct{}{}
	for serviceName, resources := range resourcesByService {
		if serviceName == labelsServiceName || serviceName == annotationsServiceName {
			continue
		}
		for _, resource := range resources {
			for _, targetID := range metadataPatchTargetIDs(resource) {
				targetIDs[targetID] = struct{}{}
			}
			for _, targetKey := range metadataPatchFallbackTargetKeys(resource) {
				targetObjectKeys[targetKey] = struct{}{}
			}
		}
	}
	if len(targetIDs) == 0 && len(targetObjectKeys) == 0 {
		return resourcesByService
	}
	ambiguousObjectKeys := metadataPatchAmbiguousObjectKeys(resourcesByService)

	for _, serviceName := range []string{labelsServiceName, annotationsServiceName} {
		resources, ok := resourcesByService[serviceName]
		if !ok {
			continue
		}
		filtered := resources[:0]
		for _, resource := range resources {
			if resource.InstanceState == nil {
				filtered = append(filtered, resource)
				continue
			}
			if _, duplicate := targetIDs[resource.InstanceState.ID]; duplicate {
				continue
			}
			if targetKey, _, ok := metadataPatchObjectKeyAndAPIVersionFromID(resource.InstanceState.ID); ok {
				if _, ambiguous := ambiguousObjectKeys[targetKey]; ambiguous {
					filtered = append(filtered, resource)
					continue
				}
				if _, duplicate := targetObjectKeys[targetKey]; duplicate {
					continue
				}
			}
			filtered = append(filtered, resource)
		}
		if len(filtered) == 0 {
			delete(resourcesByService, serviceName)
			continue
		}
		resourcesByService[serviceName] = filtered
	}
	return resourcesByService
}

func metadataPatchAmbiguousObjectKeys(resourcesByService map[string][]terraformutils.Resource) map[string]struct{} {
	apiVersionsByObjectKey := map[string]map[string]struct{}{}
	for _, serviceName := range []string{labelsServiceName, annotationsServiceName} {
		for _, resource := range resourcesByService[serviceName] {
			if resource.InstanceState == nil {
				continue
			}
			objectKey, apiVersion, ok := metadataPatchObjectKeyAndAPIVersionFromID(resource.InstanceState.ID)
			if !ok {
				continue
			}
			if _, ok := apiVersionsByObjectKey[objectKey]; !ok {
				apiVersionsByObjectKey[objectKey] = map[string]struct{}{}
			}
			apiVersionsByObjectKey[objectKey][apiVersion] = struct{}{}
		}
	}

	ambiguousObjectKeys := map[string]struct{}{}
	for objectKey, apiVersions := range apiVersionsByObjectKey {
		if len(apiVersions) > 1 {
			ambiguousObjectKeys[objectKey] = struct{}{}
		}
	}
	return ambiguousObjectKeys
}

func isDefaultServiceAccountDuplicate(resource terraformutils.Resource, defaultServiceAccountIDs map[string]struct{}) bool {
	if resource.InstanceInfo == nil || resource.InstanceState == nil {
		return false
	}
	if resource.InstanceInfo.Type != "kubernetes_service_account" && resource.InstanceInfo.Type != "kubernetes_service_account_v1" {
		return false
	}
	_, ok := defaultServiceAccountIDs[resource.InstanceState.ID]
	return ok
}

// InitClientAndConfig uses the KUBECONFIG environment variable to create
// a new rest client and config object based on the existing kubectl config
// and options passed from the plugin framework via environment variables
func initClientAndConfig() (*restclient.Config, clientcmd.ClientConfig, error) { //nolint
	// resolve kubeconfig location, prioritizing the --config global flag,
	// then the value of the KUBECONFIG env var (if any), and defaulting
	// to ~/.kube/config as a last resort.
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	}
	kubeconfig := filepath.Join(home, ".kube", "config")

	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if len(kubeconfigEnv) > 0 {
		kubeconfig = kubeconfigEnv
	}

	configFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONFIG")
	kubeConfigFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG")
	if len(configFile) > 0 {
		kubeconfig = configFile
	} else if len(kubeConfigFile) > 0 {
		kubeconfig = kubeConfigFile
	}

	if len(kubeconfig) == 0 {
		return nil, nil, fmt.Errorf("error initializing config. The KUBECONFIG environment variable must be defined")
	}

	config, err := configFromPath(kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("error obtaining kubectl config: %w", err)
	}
	client, err := config.ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("the provided credentials %q could not be used: %w", kubeconfig, err)
	}

	err = applyGlobalOptionsToConfig(client)
	if err != nil {
		return nil, nil, fmt.Errorf("error processing global plugin options: %w", err)
	}

	return client, config, nil
}

func configFromPath(path string) (clientcmd.ClientConfig, error) {
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	credentials, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("the provided credentials %q could not be loaded: %w", path, err)
	}

	overrides := &clientcmd.ConfigOverrides{
		Context: clientcmdapi.Context{
			Namespace: os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_NAMESPACE"),
		},
	}

	var cfg clientcmd.ClientConfig
	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")
	if len(context) > 0 {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		cfg = clientcmd.NewNonInteractiveClientConfig(*credentials, context, overrides, rules)
	} else {
		cfg = clientcmd.NewDefaultClientConfig(*credentials, overrides)
	}

	return cfg, nil
}

func applyGlobalOptionsToConfig(config *restclient.Config) error {
	// impersonation config
	impersonateUser := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_AS")
	if len(impersonateUser) > 0 {
		config.Impersonate.UserName = impersonateUser
	}

	impersonateGroup := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_AS_GROUP")
	if len(impersonateGroup) > 0 {
		impersonateGroupJSON := []string{}
		err := json.Unmarshal([]byte(impersonateGroup), &impersonateGroupJSON)
		if err != nil {
			return fmt.Errorf("error parsing global option %q: %w", "--as-group", err)
		}
		if len(impersonateGroupJSON) > 0 {
			config.Impersonate.Groups = impersonateGroupJSON
		}
	}

	// tls config
	caFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CERTIFICATE_AUTHORITY")
	if len(caFile) > 0 {
		config.CAFile = caFile
	}

	clientCertFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLIENT_CERTIFICATE")
	if len(clientCertFile) > 0 {
		config.CertFile = clientCertFile
	}

	clientKey := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CLIENT_KEY")
	if len(clientKey) > 0 {
		config.KeyFile = clientKey
	}

	// user / misc request config
	requestTimeout := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_REQUEST_TIMEOUT")
	if len(requestTimeout) > 0 {
		t, err := time.ParseDuration(requestTimeout)
		if err != nil {
			return fmt.Errorf("%w", err)
		}
		config.Timeout = t
	}

	server := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_SERVER")
	if len(server) > 0 {
		config.ServerName = server
	}

	token := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_TOKEN")
	if len(token) > 0 {
		config.BearerToken = token
	}

	username := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_USERNAME")
	if len(username) > 0 {
		config.Username = username
	}

	password := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_PASSWORD")
	if len(password) > 0 {
		config.Password = password
	}

	return nil
}
