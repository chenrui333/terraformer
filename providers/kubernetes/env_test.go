// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"errors"
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat"
	"github.com/chenrui333/terraformer/terraformutils/tfcompat/configschema"
	"github.com/zclconf/go-cty/cty"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestEnvInitResources(t *testing.T) {
	deployment := envTestDeployment("app", "default")
	ownedDeployment := envTestDeployment("owned", "default")
	ownedDeployment.SetOwnerReferences([]metav1.OwnerReference{{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "owner",
		UID:        "owner-uid",
	}})
	cronJob := envTestCronJob("nightly", "jobs")

	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
			{Group: "batch", Version: "v1", Resource: "cronjobs"}:   "CronJobList",
		},
		deployment,
		ownedDeployment,
		cronJob,
	)
	service := &Env{TerraformType: envTerraformType}

	if err := service.initResources(client, envTestAPIResources()); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 6 {
		t.Fatalf("Resources len = %d, want 6", len(service.Resources))
	}
	resources := resourcesByName(service.Resources)
	if _, ok := resources["tfer--env-002F-apps-002F-v1-002F-Deployment-002F-default-002F-app-002F-container-002F-app-002F-DUPLICATE"]; ok {
		t.Fatalf("duplicate env name was imported")
	}
	app := resources["tfer--env-002F-apps-002F-v1-002F-Deployment-002F-default-002F-app-002F-container-002F-app-002F-PLAIN"]
	if app.InstanceInfo.Type != envTerraformType {
		t.Fatalf("resource type = %q, want %q", app.InstanceInfo.Type, envTerraformType)
	}
	for key, want := range map[string]string{
		"id":                   "apiVersion=apps/v1,kind=Deployment,namespace=default,name=app",
		"api_version":          "apps/v1",
		"kind":                 "Deployment",
		"metadata.#":           "1",
		"metadata.0.name":      "app",
		"metadata.0.namespace": "default",
		"container":            "app",
		"field_manager":        envFieldManager("apps/v1", "Deployment", "default", "app", envContainer{Name: "app"}, "PLAIN", true),
		"env.#":                "1",
		"env.0.name":           "PLAIN",
		"env.0.value":          "demo",
		"env.0.value_from.#":   "0",
	} {
		if got := app.InstanceState.Attributes[key]; got != want {
			t.Fatalf("attribute %q = %q, want %q", key, got, want)
		}
	}
	if len(app.AllowEmptyValues) != 1 || app.AllowEmptyValues[0] != envAllowEmptyPattern {
		t.Fatalf("AllowEmptyValues = %#v, want %#v", app.AllowEmptyValues, []string{envAllowEmptyPattern})
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(app.InstanceState.Attributes, envTestBlock().ImpliedType()); err != nil {
		t.Fatalf("env flatmap does not decode into provider schema: %v", err)
	}

	secret := resources["tfer--env-002F-apps-002F-v1-002F-Deployment-002F-default-002F-app-002F-container-002F-app-002F-FROM_SECRET"]
	for key, want := range map[string]string{
		"env.0.name":         "FROM_SECRET",
		"env.0.value_from.#": "1",
		"env.0.value_from.0.config_map_key_ref.#":      "0",
		"env.0.value_from.0.field_ref.#":               "0",
		"env.0.value_from.0.resource_field_ref.#":      "0",
		"env.0.value_from.0.secret_key_ref.#":          "1",
		"env.0.value_from.0.secret_key_ref.0.name":     "app-secret",
		"env.0.value_from.0.secret_key_ref.0.key":      "password",
		"env.0.value_from.0.secret_key_ref.0.optional": "true",
	} {
		if got := secret.InstanceState.Attributes[key]; got != want {
			t.Fatalf("secret attribute %q = %q, want %q", key, got, want)
		}
	}
	if _, err := tfcompat.HCL2ValueFromFlatmap(secret.InstanceState.Attributes, envTestBlock().ImpliedType()); err != nil {
		t.Fatalf("secret env flatmap does not decode into provider schema: %v", err)
	}

	initContainer := resources["tfer--env-002F-apps-002F-v1-002F-Deployment-002F-default-002F-app-002F-init_container-002F-init-002F-FROM_CONFIG"]
	if initContainer.InstanceState.Attributes["init_container"] != "init" {
		t.Fatalf("init_container = %q, want init", initContainer.InstanceState.Attributes["init_container"])
	}
	if initContainer.InstanceState.Attributes["env.0.value_from.0.config_map_key_ref.0.name"] != "app-config" {
		t.Fatalf("init config map ref was not flattened")
	}

	cron := resources["tfer--env-002F-batch-002F-v1-002F-CronJob-002F-jobs-002F-nightly-002F-container-002F-job-002F-SCHEDULE"]
	if cron.InstanceState.Attributes["id"] != "apiVersion=batch/v1,kind=CronJob,namespace=jobs,name=nightly" {
		t.Fatalf("cron ID = %q, want CronJob import ID", cron.InstanceState.Attributes["id"])
	}
}

func TestEnvInitResourcesSkipsListFailures(t *testing.T) {
	deployment := envTestDeployment("app", "default")
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
			{Group: "apps", Version: "v1", Resource: "daemonsets"}:  "DaemonSetList",
		},
		deployment,
	)
	client.PrependReactor("list", "daemonsets", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})
	service := &Env{TerraformType: envTerraformType}

	if err := service.initResources(client, []*metav1.APIResourceList{{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			{Name: "daemonsets", Kind: "DaemonSet", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
		},
	}}); err != nil {
		t.Fatalf("initResources() error = %v", err)
	}

	if len(service.Resources) != 5 {
		t.Fatalf("Resources len = %d, want 5", len(service.Resources))
	}
	for _, resource := range service.Resources {
		if resource.InstanceState.ID != "apiVersion=apps/v1,kind=Deployment,namespace=default,name=app" {
			t.Fatalf("resource ID = %q, want deployment ID", resource.InstanceState.ID)
		}
	}
}

func TestEnvSupportsResourceRequiresWorkloadAndManageableVerbs(t *testing.T) {
	tests := []struct {
		name     string
		resource metav1.APIResource
		want     bool
	}{
		{
			name:     "manageable deployment",
			resource: metav1.APIResource{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			want:     true,
		},
		{
			name:     "pod is not a pod template workload",
			resource: metav1.APIResource{Name: "pods", Kind: "Pod", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
		},
		{
			name:     "list only",
			resource: metav1.APIResource{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"list"}},
		},
		{
			name:     "subresource",
			resource: metav1.APIResource{Name: "deployments/status", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
		},
		{
			name:     "non workload",
			resource: metav1.APIResource{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envSupportsResource(tt.resource); got != tt.want {
				t.Fatalf("envSupportsResource() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestAddEnvService(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	envResources := map[kubernetesResourceID]struct{}{
		{group: "apps", version: "v1", kind: "Deployment"}: {},
	}

	addEnvService(resources, envResources, func(name string) bool {
		return name == envTerraformType
	})

	service, ok := resources[envServiceName]
	if !ok {
		t.Fatalf("resources[%q] was not registered", envServiceName)
	}
	env, ok := service.(*Env)
	if !ok {
		t.Fatalf("service type = %T, want *Env", service)
	}
	if env.TerraformType != envTerraformType {
		t.Fatalf("TerraformType = %q, want %q", env.TerraformType, envTerraformType)
	}
}

func TestAddEnvServiceRequiresProviderType(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	envResources := map[kubernetesResourceID]struct{}{
		{group: "apps", version: "v1", kind: "Deployment"}: {},
	}

	addEnvService(resources, envResources, func(string) bool {
		return false
	})

	if _, ok := resources[envServiceName]; ok {
		t.Fatalf("resources[%q] was registered without provider type support", envServiceName)
	}
}

func TestAddEnvServiceRequiresWorkloadAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}
	envResources := map[kubernetesResourceID]struct{}{
		{version: "v1", kind: "ConfigMap"}: {},
	}

	addEnvService(resources, envResources, func(name string) bool {
		return name == envTerraformType
	})

	if _, ok := resources[envServiceName]; ok {
		t.Fatalf("resources[%q] was registered without workload API support", envServiceName)
	}
}

func TestAddEnvServiceRequiresManageableWorkloadAPI(t *testing.T) {
	resources := map[string]terraformutils.ServiceGenerator{}

	addEnvService(resources, map[kubernetesResourceID]struct{}{}, func(name string) bool {
		return name == envTerraformType
	})

	if _, ok := resources[envServiceName]; ok {
		t.Fatalf("resources[%q] was registered without manageable workload API support", envServiceName)
	}
}

func TestPostProcessImportResourcesRemovesOverlappingEnv(t *testing.T) {
	provider := KubernetesProvider{}
	fullDeployment := terraformutils.NewSimpleResource(
		"default/app",
		"default/app",
		"kubernetes_deployment_v1",
		"kubernetes",
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"deployments": {fullDeployment},
		envServiceName: {
			envTestResource("apiVersion=apps/v1,kind=Deployment,namespace=default,name=app", "app"),
			envTestResource("apiVersion=apps/v1,kind=Deployment,namespace=default,name=app", "sidecar"),
			envTestResource("apiVersion=apps/v1,kind=Deployment,namespace=default,name=other", "app"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	assertResourceIDs(t, got[envServiceName], []string{"apiVersion=apps/v1,kind=Deployment,namespace=default,name=other"})
}

func TestPostProcessImportResourcesRemovesManifestEnvOverlap(t *testing.T) {
	provider := KubernetesProvider{}
	manifest := terraformutils.NewSimpleResource(
		"apiVersion=apps/v1,kind=Deployment,namespace=default,name=app",
		"apps/v1/Deployment/default/app",
		manifestTerraformResourceName,
		"kubernetes",
		nil,
	)
	resourcesByService := map[string][]terraformutils.Resource{
		"apps/v1/deployments": {manifest},
		envServiceName: {
			envTestResource("apiVersion=apps/v1,kind=Deployment,namespace=default,name=app", "app"),
		},
	}

	got := provider.PostProcessImportResources(resourcesByService)

	if _, ok := got[envServiceName]; ok {
		t.Fatalf("resources[%q] was not removed after manifest overlap", envServiceName)
	}
}

func envTestDeployment(name, namespace string) *unstructured.Unstructured {
	deployment := newUnstructured("apps/v1", "Deployment", name, namespace)
	_ = unstructured.SetNestedSlice(deployment.Object, []interface{}{
		map[string]interface{}{
			"name": "app",
			"env": []interface{}{
				map[string]interface{}{"name": "DUPLICATE", "value": "ignored"},
				map[string]interface{}{"name": "DUPLICATE", "value": "still-ignored"},
				map[string]interface{}{"name": "PLAIN", "value": "demo"},
				map[string]interface{}{"name": "EMPTY", "value": ""},
				map[string]interface{}{
					"name": "FROM_SECRET",
					"valueFrom": map[string]interface{}{
						"secretKeyRef": map[string]interface{}{"name": "app-secret", "key": "password", "optional": true},
					},
				},
				map[string]interface{}{
					"name": "POD_NAME",
					"valueFrom": map[string]interface{}{
						"fieldRef": map[string]interface{}{"apiVersion": "v1", "fieldPath": "metadata.name"},
					},
				},
			},
		},
		map[string]interface{}{"name": "sidecar"},
	}, "spec", "template", "spec", "containers")
	_ = unstructured.SetNestedSlice(deployment.Object, []interface{}{
		map[string]interface{}{
			"name": "init",
			"env": []interface{}{
				map[string]interface{}{
					"name": "FROM_CONFIG",
					"valueFrom": map[string]interface{}{
						"configMapKeyRef": map[string]interface{}{"name": "app-config", "key": "setting"},
					},
				},
			},
		},
	}, "spec", "template", "spec", "initContainers")
	return deployment
}

func envTestCronJob(name, namespace string) *unstructured.Unstructured {
	cronJob := newUnstructured("batch/v1", "CronJob", name, namespace)
	_ = unstructured.SetNestedSlice(cronJob.Object, []interface{}{
		map[string]interface{}{
			"name": "job",
			"env": []interface{}{
				map[string]interface{}{"name": "SCHEDULE", "value": "nightly"},
			},
		},
	}, "spec", "jobTemplate", "spec", "template", "spec", "containers")
	return cronJob
}

func TestEnvEntriesSkipsDuplicateNames(t *testing.T) {
	got := envEntries([]interface{}{
		map[string]interface{}{"name": "DUPLICATE", "value": "first"},
		map[string]interface{}{"name": "OTHER", "value": "middle"},
		map[string]interface{}{"name": "DUPLICATE", "value": "last"},
	})

	if len(got) != 1 {
		t.Fatalf("env entries len = %d, want 1", len(got))
	}
	if got[0]["name"] != "OTHER" || got[0]["value"] != "middle" {
		t.Fatalf("kept env = %#v, want OTHER=middle", got[0])
	}
}

func envTestAPIResources() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
				{Name: "deployments/status", Kind: "Deployment", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			},
		},
		{
			GroupVersion: "batch/v1",
			APIResources: []metav1.APIResource{
				{Name: "cronjobs", Kind: "CronJob", Namespaced: true, Verbs: []string{"get", "list", "patch"}},
			},
		},
	}
}

func envTestResource(id, container string) terraformutils.Resource {
	return terraformutils.NewResource(
		id,
		id+"/"+container,
		envTerraformType,
		"kubernetes",
		map[string]string{"id": id, "container": container},
		nil,
		nil,
	)
}

func resourcesByName(resources []terraformutils.Resource) map[string]terraformutils.Resource {
	byName := map[string]terraformutils.Resource{}
	for _, resource := range resources {
		byName[resource.ResourceName] = resource
	}
	return byName
}

func envTestBlock() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {
				Type:     cty.String,
				Computed: true,
			},
			"api_version": {
				Type:     cty.String,
				Required: true,
			},
			"kind": {
				Type:     cty.String,
				Required: true,
			},
			"container": {
				Type:     cty.String,
				Optional: true,
			},
			"init_container": {
				Type:     cty.String,
				Optional: true,
			},
			"field_manager": {
				Type:     cty.String,
				Optional: true,
			},
			"force": {
				Type:     cty.Bool,
				Optional: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"metadata": {
				Nesting:  configschema.NestingList,
				MinItems: 1,
				MaxItems: 1,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"name": {
							Type:     cty.String,
							Required: true,
						},
						"namespace": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
			"env": {
				Nesting:  configschema.NestingList,
				MinItems: 1,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"name": {
							Type:     cty.String,
							Required: true,
						},
						"value": {
							Type:     cty.String,
							Optional: true,
						},
					},
					BlockTypes: map[string]*configschema.NestedBlock{
						"value_from": {
							Nesting:  configschema.NestingList,
							MaxItems: 1,
							Block:    envValueFromTestBlock(),
						},
					},
				},
			},
		},
	}
}

func envValueFromTestBlock() configschema.Block {
	refBlock := func(attributes map[string]*configschema.Attribute) configschema.Block {
		return configschema.Block{Attributes: attributes}
	}
	return configschema.Block{
		BlockTypes: map[string]*configschema.NestedBlock{
			"config_map_key_ref": {
				Nesting:  configschema.NestingList,
				MaxItems: 1,
				Block: refBlock(map[string]*configschema.Attribute{
					"key":      {Type: cty.String, Optional: true},
					"name":     {Type: cty.String, Optional: true},
					"optional": {Type: cty.Bool, Optional: true},
				}),
			},
			"field_ref": {
				Nesting:  configschema.NestingList,
				MaxItems: 1,
				Block: refBlock(map[string]*configschema.Attribute{
					"api_version": {Type: cty.String, Optional: true},
					"field_path":  {Type: cty.String, Optional: true},
				}),
			},
			"resource_field_ref": {
				Nesting:  configschema.NestingList,
				MaxItems: 1,
				Block: refBlock(map[string]*configschema.Attribute{
					"container_name": {Type: cty.String, Optional: true},
					"divisor":        {Type: cty.String, Optional: true},
					"resource":       {Type: cty.String, Required: true},
				}),
			},
			"secret_key_ref": {
				Nesting:  configschema.NestingList,
				MaxItems: 1,
				Block: refBlock(map[string]*configschema.Attribute{
					"key":      {Type: cty.String, Optional: true},
					"name":     {Type: cty.String, Optional: true},
					"optional": {Type: cty.Bool, Optional: true},
				}),
			},
		},
	}
}
