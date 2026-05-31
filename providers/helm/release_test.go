// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"helm.sh/helm/v4/pkg/action"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	helmreleasecommon "helm.sh/helm/v4/pkg/release/common"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/client-go/rest"
)

type fakeReleaseDiscovery struct {
	listed        []*helmrelease.Release
	getByID       map[string]*helmrelease.Release
	listWasCalled bool
	gotIDs        []string
}

func (d *fakeReleaseDiscovery) GetRelease(namespace, name string) (*helmrelease.Release, error) {
	id := releaseImportID{Namespace: namespace, Name: name}.String()
	d.gotIDs = append(d.gotIDs, id)
	release, ok := d.getByID[id]
	if !ok {
		return nil, fmt.Errorf("missing release %s", id)
	}
	return release, nil
}

func (d *fakeReleaseDiscovery) ListReleases() ([]*helmrelease.Release, error) {
	d.listWasCalled = true
	return d.listed, nil
}

func clearHelmDiscoveryKubeEnv(t *testing.T) {
	for _, env := range []string{
		"KUBE_CONFIG_PATH",
		"KUBE_CONFIG_PATHS",
		"KUBECONFIG",
		"KUBE_CTX",
		"KUBE_CTX_AUTH_INFO",
		"KUBE_CTX_CLUSTER",
		"KUBE_HOST",
		"KUBE_TOKEN",
		"KUBE_USER",
		"KUBE_PASSWORD",
		"KUBE_INSECURE",
		"KUBE_TLS_SERVER_NAME",
		"KUBE_CLUSTER_CA_CERT_DATA",
		"KUBE_CLIENT_CERT_DATA",
		"KUBE_CLIENT_KEY_DATA",
		"KUBE_PROXY_URL",
		"HELM_KUBECONTEXT",
		"HELM_KUBEAPISERVER",
		"HELM_KUBETOKEN",
		"HELM_KUBECAFILE",
		"HELM_KUBEINSECURE_SKIP_TLS_VERIFY",
		"HELM_KUBETLS_SERVER_NAME",
	} {
		t.Setenv(env, "")
	}
}

func disableDefaultKubeconfig(t *testing.T) {
	originalDefaultKubeConfigPath := helmDefaultKubeConfigPath
	missingKubeconfig := filepath.Join(t.TempDir(), "missing-kubeconfig")
	helmDefaultKubeConfigPath = func() string {
		return missingKubeconfig
	}
	t.Cleanup(func() {
		helmDefaultKubeConfigPath = originalDefaultKubeConfigPath
	})
}

func TestHelmReleaseDiscoveryListActionUsesAllNamespacesStorage(t *testing.T) {
	var gotNamespace string
	discovery := &helmReleaseDiscovery{
		actionConfigFactory: func(namespace string) (*action.Configuration, error) {
			gotNamespace = namespace
			return &action.Configuration{}, nil
		},
	}

	list, err := discovery.newListAction()
	if err != nil {
		t.Fatalf("newListAction() error = %v", err)
	}
	if gotNamespace != "" {
		t.Fatalf("list action config namespace = %q, want empty namespace for all namespaces", gotNamespace)
	}
	if !list.All || !list.AllNamespaces {
		t.Fatalf("list flags All=%v AllNamespaces=%v, want both true", list.All, list.AllNamespaces)
	}
	if list.StateMask != action.ListAll {
		t.Fatalf("list StateMask = %v, want ListAll", list.StateMask)
	}
}

func TestHelmReleaseDiscoveryMirrorsProviderKubeEnv(t *testing.T) {
	t.Setenv("KUBE_CONFIG_PATH", "/tmp/provider-kubeconfig")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBE_CTX", "provider-context")
	t.Setenv("HELM_KUBECONTEXT", "")

	discovery := newHelmReleaseDiscovery()

	if got := os.Getenv("KUBECONFIG"); got != "/tmp/provider-kubeconfig" {
		t.Fatalf("KUBECONFIG = %q, want provider kubeconfig path", got)
	}
	if got := os.Getenv("HELM_KUBECONTEXT"); got != "provider-context" {
		t.Fatalf("HELM_KUBECONTEXT = %q, want provider context", got)
	}
	if discovery.settings.KubeContext != "provider-context" {
		t.Fatalf("discovery kube context = %q, want provider-context", discovery.settings.KubeContext)
	}
}

func TestHelmReleaseDiscoverySupportsProviderOnlyKubeAuth(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		env    map[string]string
		assert func(t *testing.T, config *rest.Config)
	}{
		{
			name: "token and CA data",
			env: map[string]string{
				"KUBE_TOKEN":                "provider-token",
				"KUBE_CLUSTER_CA_CERT_DATA": "ca-data",
				"KUBE_PROXY_URL":            "http://proxy.example.test:8080",
			},
			assert: func(t *testing.T, config *rest.Config) {
				if config.BearerToken != "provider-token" {
					t.Fatalf("REST bearer token = %q, want provider token", config.BearerToken)
				}
				if string(config.CAData) != "ca-data" {
					t.Fatalf("REST CAData = %q, want provider CA data", string(config.CAData))
				}
				if config.Proxy == nil {
					t.Fatal("REST proxy is nil, want provider proxy URL")
				}
				proxyURL, err := config.Proxy(&http.Request{})
				if err != nil {
					t.Fatalf("REST proxy error = %v", err)
				}
				if proxyURL.String() != "http://proxy.example.test:8080" {
					t.Fatalf("REST proxy URL = %q, want provider proxy URL", proxyURL.String())
				}
			},
		},
		{
			name: "client certificate",
			env: map[string]string{
				"KUBE_CLIENT_CERT_DATA": "client-cert-data",
				"KUBE_CLIENT_KEY_DATA":  "client-key-data",
			},
			assert: func(t *testing.T, config *rest.Config) {
				if string(config.CertData) != "client-cert-data" {
					t.Fatalf("REST CertData = %q, want provider client cert data", string(config.CertData))
				}
				if string(config.KeyData) != "client-key-data" {
					t.Fatalf("REST KeyData = %q, want provider client key data", string(config.KeyData))
				}
			},
		},
		{
			name: "basic auth",
			env: map[string]string{
				"KUBE_USER":     "provider-user",
				"KUBE_PASSWORD": "provider-password",
			},
			assert: func(t *testing.T, config *rest.Config) {
				if config.Username != "provider-user" {
					t.Fatalf("REST username = %q, want provider user", config.Username)
				}
				if config.Password != "provider-password" {
					t.Fatalf("REST password = %q, want provider password", config.Password)
				}
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			clearHelmDiscoveryKubeEnv(t)
			disableDefaultKubeconfig(t)
			t.Setenv("KUBE_HOST", "https://example.test")
			t.Setenv("KUBE_CTX_AUTH_INFO", "provider-user-context")
			t.Setenv("KUBE_CTX_CLUSTER", "provider-cluster-context")
			for env, value := range testCase.env {
				t.Setenv(env, value)
			}

			discovery := newHelmReleaseDiscovery()
			restConfig, err := discovery.restClientGetter.ToRESTConfig()
			if err != nil {
				t.Fatalf("ToRESTConfig() error = %v", err)
			}
			if restConfig.Host != "https://example.test" {
				t.Fatalf("REST host = %q, want provider host", restConfig.Host)
			}
			testCase.assert(t, restConfig)
		})
	}
}

func TestHelmReleaseDiscoverySupportsProviderKubeContextOverrides(t *testing.T) {
	clearHelmDiscoveryKubeEnv(t)
	kubeconfig := filepath.Join(t.TempDir(), "config")
	contents := []byte(`apiVersion: v1
kind: Config
clusters:
- name: base-cluster
  cluster:
    server: https://base.example.test
    insecure-skip-tls-verify: true
- name: override-cluster
  cluster:
    server: https://override.example.test
    insecure-skip-tls-verify: true
users:
- name: base-user
  user:
    token: base-token
- name: override-user
  user:
    token: override-token
contexts:
- name: base
  context:
    cluster: base-cluster
    user: base-user
current-context: base
`)
	if err := os.WriteFile(kubeconfig, contents, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("KUBE_CONFIG_PATH", kubeconfig)
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBE_CTX", "base")
	t.Setenv("KUBE_CTX_AUTH_INFO", "override-user")
	t.Setenv("KUBE_CTX_CLUSTER", "override-cluster")

	discovery := newHelmReleaseDiscovery()
	restConfig, err := discovery.restClientGetter.ToRESTConfig()
	if err != nil {
		t.Fatalf("ToRESTConfig() error = %v", err)
	}
	if restConfig.Host != "https://override.example.test" {
		t.Fatalf("REST host = %q, want override cluster host", restConfig.Host)
	}
	if restConfig.BearerToken != "override-token" {
		t.Fatalf("REST bearer token = %q, want override user token", restConfig.BearerToken)
	}
}

func TestReleaseImportIDConstruction(t *testing.T) {
	id := releaseImportID{Namespace: "default", Name: "nginx"}
	if got := id.String(); got != "default/nginx" {
		t.Fatalf("release import ID = %q, want default/nginx", got)
	}

	parsed, err := parseReleaseImportID("kube-system/metrics-server")
	if err != nil {
		t.Fatalf("parseReleaseImportID() error = %v", err)
	}
	if parsed.Namespace != "kube-system" || parsed.Name != "metrics-server" {
		t.Fatalf("parsed ID = %#v, want kube-system/metrics-server", parsed)
	}
}

func TestReleaseImportIDRequiresNamespaceAndName(t *testing.T) {
	for _, input := range []string{"nginx", "default/", "/nginx", "default/nginx/extra"} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseReleaseImportID(input); err == nil {
				t.Fatalf("parseReleaseImportID(%q) returned nil error", input)
			}
		})
	}
}

func TestReleaseExactFilterParsing(t *testing.T) {
	generator := &ReleaseGenerator{}
	generator.ParseFilters([]string{"release=default/nginx"})

	ids, err := generator.releaseIDFilters()
	if err != nil {
		t.Fatalf("releaseIDFilters() error = %v", err)
	}
	want := []releaseImportID{{Namespace: "default", Name: "nginx"}}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("release ID filters = %#v, want %#v", ids, want)
	}
}

func TestReleaseExactFilterParsingMultipleIDs(t *testing.T) {
	generator := &ReleaseGenerator{}
	generator.ParseFilters([]string{"release=default/nginx:kube-system/metrics-server"})

	ids, err := generator.releaseIDFilters()
	if err != nil {
		t.Fatalf("releaseIDFilters() error = %v", err)
	}
	want := []releaseImportID{
		{Namespace: "default", Name: "nginx"},
		{Namespace: "kube-system", Name: "metrics-server"},
	}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("release ID filters = %#v, want %#v", ids, want)
	}
}

func TestReleaseStatusSkipPredicates(t *testing.T) {
	testCases := map[helmreleasecommon.Status]bool{
		helmreleasecommon.StatusDeployed:        true,
		helmreleasecommon.StatusSuperseded:      false,
		helmreleasecommon.StatusUninstalled:     false,
		helmreleasecommon.StatusUninstalling:    false,
		helmreleasecommon.StatusPendingInstall:  false,
		helmreleasecommon.StatusPendingUpgrade:  false,
		helmreleasecommon.StatusPendingRollback: false,
		helmreleasecommon.StatusFailed:          false,
		helmreleasecommon.StatusUnknown:         false,
	}

	for status, want := range testCases {
		t.Run(status.String(), func(t *testing.T) {
			if got := isImportableReleaseStatus(status); got != want {
				t.Fatalf("isImportableReleaseStatus(%q) = %v, want %v", status, got, want)
			}
		})
	}
}

func TestLatestReleaseSelectionRequiresLatestRevisionToBeDeployed(t *testing.T) {
	releases := []*helmrelease.Release{
		testRelease("api", "default", 1, helmreleasecommon.StatusDeployed),
		testRelease("api", "default", 2, helmreleasecommon.StatusFailed),
		testRelease("web", "default", 1, helmreleasecommon.StatusSuperseded),
		testRelease("web", "default", 2, helmreleasecommon.StatusDeployed),
		testRelease("worker", "jobs", 1, helmreleasecommon.StatusPendingUpgrade),
	}

	selected := selectLatestImportableReleases(releases)
	if len(selected) != 1 {
		t.Fatalf("selected releases = %d, want 1", len(selected))
	}
	if selected[0].Namespace != "default" || selected[0].Name != "web" || selected[0].Version != 2 {
		t.Fatalf("selected release = %#v, want default/web revision 2", selected[0])
	}
}

func TestReleaseResourceSeedsSafeFields(t *testing.T) {
	release := testRelease("nginx", "default", 3, helmreleasecommon.StatusDeployed)
	release.Chart.Metadata.Name = "nginx-chart"
	release.Chart.Metadata.Version = "1.2.3"
	release.Info.Description = "Install complete"

	resources := createReleaseResources([]*helmrelease.Release{release})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	resource := resources[0]
	if resource.InstanceInfo.Type != "helm_release" {
		t.Fatalf("resource type = %q, want helm_release", resource.InstanceInfo.Type)
	}
	if resource.InstanceState.ID != "default/nginx" {
		t.Fatalf("resource ID = %q, want default/nginx", resource.InstanceState.ID)
	}
	wantAttributes := map[string]string{
		"name":        "nginx",
		"namespace":   "default",
		"chart":       "nginx-chart",
		"version":     "1.2.3",
		"description": "Install complete",
	}
	if !reflect.DeepEqual(resource.InstanceState.Attributes, wantAttributes) {
		t.Fatalf("resource attributes = %#v, want %#v", resource.InstanceState.Attributes, wantAttributes)
	}
}

func TestReleaseResourceLabelCollisions(t *testing.T) {
	releases := []*helmrelease.Release{
		testRelease("c", "a_b", 1, helmreleasecommon.StatusDeployed),
		testRelease("b_c", "a", 1, helmreleasecommon.StatusDeployed),
	}

	resources := createReleaseResources(releases)
	if len(resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(resources))
	}

	resourcesByID := map[string]terraformutils.Resource{}
	resourceNames := map[string]struct{}{}
	for _, resource := range resources {
		if _, exists := resourceNames[resource.ResourceName]; exists {
			t.Fatalf("duplicate resource name %q in %#v", resource.ResourceName, resources)
		}
		resourceNames[resource.ResourceName] = struct{}{}
		resourcesByID[resource.InstanceState.ID] = resource
	}

	want := map[string]map[string]string{
		"a_b/c": {
			"namespace": "a_b",
			"name":      "c",
		},
		"a/b_c": {
			"namespace": "a",
			"name":      "b_c",
		},
	}
	for id, attributes := range want {
		resource, ok := resourcesByID[id]
		if !ok {
			t.Fatalf("missing resource ID %q in %#v", id, resourcesByID)
		}
		for key, value := range attributes {
			if got := resource.InstanceState.Attributes[key]; got != value {
				t.Fatalf("resource %q attribute %q = %q, want %q", id, key, got, value)
			}
		}
	}
}

func TestReleaseResourceDoesNotExportValuesSecretsOrManifests(t *testing.T) {
	release := testRelease("payments", "apps", 1, helmreleasecommon.StatusDeployed)
	release.Config = map[string]interface{}{
		"adminPassword": "should-not-export",
	}
	release.Manifest = "apiVersion: v1\nkind: Secret\ndata:\n  password: should-not-export\n"
	release.Chart.Values = map[string]interface{}{
		"token": "should-not-export",
	}
	release.Info.Description = "rotated secret token"

	resources := createReleaseResources([]*helmrelease.Release{release})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	attributes := resources[0].InstanceState.Attributes
	for _, key := range []string{"values", "set", "set_list", "set_sensitive", "set_wo", "manifest", "repository"} {
		if _, ok := attributes[key]; ok {
			t.Fatalf("attributes unexpectedly exported %q: %#v", key, attributes)
		}
	}
	for key, value := range attributes {
		if value == "should-not-export" {
			t.Fatalf("attribute %q exported sensitive value", key)
		}
	}
	if _, ok := attributes["description"]; ok {
		t.Fatalf("unsafe description was exported: %#v", attributes)
	}
}

func TestReleasePostConvertHookStripsRefreshedValues(t *testing.T) {
	resource := terraformutils.NewResource(
		"apps/payments",
		"release_apps_payments",
		helmReleaseResourceType,
		helmProviderName,
		map[string]string{
			"name":                           "payments",
			"namespace":                      "apps",
			"metadata.chart":                 "payments-chart",
			"metadata.values":                "{\"adminPassword\":\"should-not-export\"}",
			"metadata.notes":                 "token: should-not-export",
			"metadata.0.values":              "{\"token\":\"should-not-export\"}",
			"metadata.0.notes":               "password: should-not-export",
			"values.#":                       "1",
			"values.0":                       "adminPassword: should-not-export",
			"set.0.name":                     "adminPassword",
			"set.0.value":                    "should-not-export",
			"set_sensitive.0.name":           "token",
			"set_sensitive.0.value":          "should-not-export",
			"manifest":                       "{\"kind\":\"Secret\",\"data\":{\"password\":\"should-not-export\"}}",
			"resources.Secret.apps.payments": "{\"data\":{\"password\":\"should-not-export\"}}",
		},
		nil,
		nil,
	)
	resource.Item = map[string]interface{}{
		"name":      "payments",
		"namespace": "apps",
		"metadata": []interface{}{
			map[string]interface{}{
				"chart":  "payments-chart",
				"values": "{\"adminPassword\":\"should-not-export\"}",
				"notes":  "token: should-not-export",
			},
		},
		"values":   []interface{}{"adminPassword: should-not-export"},
		"manifest": "secret manifest should-not-export",
		"resources": map[string]interface{}{
			"Secret/apps/payments": "should-not-export",
		},
	}
	resource.InstanceState.SetTypedAttributes(json.RawMessage("{\"name\":\"payments\",\"namespace\":\"apps\",\"metadata\":{\"chart\":\"payments-chart\",\"values\":\"{\\\"adminPassword\\\":\\\"should-not-export\\\"}\",\"notes\":\"token: should-not-export\"},\"values\":[\"adminPassword: should-not-export\"],\"set\":[{\"name\":\"adminPassword\",\"value\":\"should-not-export\"}],\"set_sensitive\":[{\"name\":\"token\",\"value\":\"should-not-export\"}],\"manifest\":\"secret manifest should-not-export\",\"resources\":{\"Secret/apps/payments\":\"should-not-export\"}}"))
	generator := &ReleaseGenerator{
		Service: terraformutils.Service{
			Resources: []terraformutils.Resource{resource},
		},
	}

	if err := generator.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() error = %v", err)
	}
	updated := generator.Resources[0]
	for key, value := range updated.InstanceState.Attributes {
		if isHelmReleaseUnsafeFlatAttribute(key) {
			t.Fatalf("unsafe flat attribute %q was not removed: %#v", key, updated.InstanceState.Attributes)
		}
		if value == "should-not-export" {
			t.Fatalf("flat attribute %q retained sensitive value", key)
		}
	}
	if updated.InstanceState.Attributes["metadata.chart"] != "payments-chart" {
		t.Fatalf("metadata.chart = %q, want payments-chart", updated.InstanceState.Attributes["metadata.chart"])
	}

	assertNoHelmReleaseUnsafeItemFields(t, updated.Item)
	metadata := updated.Item["metadata"].([]interface{})[0].(map[string]interface{})
	if metadata["chart"] != "payments-chart" {
		t.Fatalf("item metadata chart = %v, want payments-chart", metadata["chart"])
	}

	var typedAttributes map[string]interface{}
	if err := json.Unmarshal(updated.InstanceState.TypedAttributes, &typedAttributes); err != nil {
		t.Fatalf("TypedAttributes unmarshal error = %v", err)
	}
	assertNoHelmReleaseUnsafeItemFields(t, typedAttributes)
	typedMetadata := typedAttributes["metadata"].(map[string]interface{})
	if typedMetadata["chart"] != "payments-chart" {
		t.Fatalf("typed metadata chart = %v, want payments-chart", typedMetadata["chart"])
	}
	if !updated.InstanceState.HasCurrentTypedAttributes() {
		t.Fatal("PostConvertHook left typed attributes out of sync with flat state")
	}
}

func TestReleaseResourceOmitsMissingChartAndRepository(t *testing.T) {
	release := testRelease("local", "default", 1, helmreleasecommon.StatusDeployed)
	release.Chart = nil

	resources := createReleaseResources([]*helmrelease.Release{release})
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	attributes := resources[0].InstanceState.Attributes
	for _, key := range []string{"chart", "version", "repository"} {
		if _, ok := attributes[key]; ok {
			t.Fatalf("attributes unexpectedly included %q: %#v", key, attributes)
		}
	}
}

func TestReleaseGeneratorInitResourcesUsesRealDiscovery(t *testing.T) {
	discovery := &fakeReleaseDiscovery{
		listed: []*helmrelease.Release{
			testRelease("nginx", "apps", 1, helmreleasecommon.StatusDeployed),
		},
	}
	generator := &ReleaseGenerator{discovery: discovery}

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	if !discovery.listWasCalled {
		t.Fatal("ListReleases was not called for broad discovery")
	}
	if len(generator.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(generator.Resources))
	}
	if generator.Resources[0].InstanceState.ID != "apps/nginx" {
		t.Fatalf("resource ID = %q, want apps/nginx", generator.Resources[0].InstanceState.ID)
	}
}

func TestReleaseGeneratorExactFiltersUseNamespaceNameGets(t *testing.T) {
	discovery := &fakeReleaseDiscovery{
		getByID: map[string]*helmrelease.Release{
			"default/nginx":              testRelease("nginx", "default", 1, helmreleasecommon.StatusDeployed),
			"kube-system/metrics-server": testRelease("metrics-server", "kube-system", 2, helmreleasecommon.StatusDeployed),
		},
	}
	generator := &ReleaseGenerator{
		Service:   terraformutils.Service{},
		discovery: discovery,
	}
	generator.ParseFilters([]string{"release=default/nginx:kube-system/metrics-server"})

	if err := generator.InitResources(); err != nil {
		t.Fatalf("InitResources() error = %v", err)
	}
	wantGets := []string{"default/nginx", "kube-system/metrics-server"}
	if !reflect.DeepEqual(discovery.gotIDs, wantGets) {
		t.Fatalf("GetRelease IDs = %#v, want %#v", discovery.gotIDs, wantGets)
	}
	if discovery.listWasCalled {
		t.Fatal("ListReleases was called for exact filters")
	}
	if len(generator.Resources) != 2 {
		t.Fatalf("resources = %d, want 2", len(generator.Resources))
	}
}

func TestReleasePostRefreshCleanupKeepsExactFilterAfterProviderIDNormalization(t *testing.T) {
	resource := terraformutils.NewResource(
		"default/nginx",
		"release_default_nginx",
		helmReleaseResourceType,
		helmProviderName,
		map[string]string{
			"name":      "nginx",
			"namespace": "default",
		},
		nil,
		nil,
	)
	resource.InstanceState.ID = "nginx"
	generator := &ReleaseGenerator{
		Service: terraformutils.Service{
			Resources: []terraformutils.Resource{resource},
		},
	}
	generator.ParseFilters([]string{"release=default/nginx"})

	generator.PostRefreshCleanup()

	if len(generator.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(generator.Resources))
	}
	if got := generator.Resources[0].InstanceState.ID; got != "nginx" {
		t.Fatalf("resource state ID = %q, want provider-normalized ID preserved", got)
	}
}

func TestReleasePostRefreshCleanupDropsExactFilterMismatchAfterProviderIDNormalization(t *testing.T) {
	resource := terraformutils.NewResource(
		"default/nginx",
		"release_default_nginx",
		helmReleaseResourceType,
		helmProviderName,
		map[string]string{
			"name":      "nginx",
			"namespace": "default",
		},
		nil,
		nil,
	)
	resource.InstanceState.ID = "nginx"
	generator := &ReleaseGenerator{
		Service: terraformutils.Service{
			Resources: []terraformutils.Resource{resource},
		},
	}
	generator.ParseFilters([]string{"release=kube-system/nginx"})

	generator.PostRefreshCleanup()

	if len(generator.Resources) != 0 {
		t.Fatalf("resources = %d, want 0", len(generator.Resources))
	}
}

func testRelease(name, namespace string, version int, status helmreleasecommon.Status) *helmrelease.Release {
	return &helmrelease.Release{
		Name:      name,
		Namespace: namespace,
		Version:   version,
		Info: &helmrelease.Info{
			Status:      status,
			Description: "Install complete",
		},
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    name,
				Version: "0.1.0",
			},
			Values: map[string]interface{}{},
		},
	}
}

func assertNoHelmReleaseUnsafeItemFields(t *testing.T, item map[string]interface{}) {
	t.Helper()
	for key := range helmReleaseUnsafeStateFields {
		if _, ok := item[key]; ok {
			t.Fatalf("unsafe item field %q was not removed: %#v", key, item)
		}
	}
	metadataValue, ok := item["metadata"]
	if !ok {
		return
	}
	metadataItems := []map[string]interface{}{}
	switch metadata := metadataValue.(type) {
	case map[string]interface{}:
		metadataItems = append(metadataItems, metadata)
	case []interface{}:
		for _, element := range metadata {
			metadataItem, ok := element.(map[string]interface{})
			if ok {
				metadataItems = append(metadataItems, metadataItem)
			}
		}
	}
	for _, metadata := range metadataItems {
		for key := range helmReleaseUnsafeMetadataFields {
			if _, ok := metadata[key]; ok {
				t.Fatalf("unsafe metadata field %q was not removed: %#v", key, metadata)
			}
		}
	}
}
