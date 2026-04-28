// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	publictypes "github.com/aws/aws-sdk-go-v2/service/ecrpublic/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestEcrAccountSettingNames(t *testing.T) {
	want := []string{"BASIC_SCAN_TYPE_VERSION", "REGISTRY_POLICY_SCOPE"}
	if len(ecrAccountSettingNames) != len(want) {
		t.Fatalf("ecrAccountSettingNames length = %d, want %d", len(ecrAccountSettingNames), len(want))
	}
	for i, settingName := range ecrAccountSettingNames {
		if settingName != want[i] {
			t.Fatalf("ecrAccountSettingNames[%d] = %q, want %q", i, settingName, want[i])
		}
	}
}

func TestEcrOptionalErrorHelpers(t *testing.T) {
	registryPolicyErr := &types.RegistryPolicyNotFoundException{}
	pullThroughErr := &types.PullThroughCacheRuleNotFoundException{}
	templateErr := &types.TemplateNotFoundException{}
	invalidParameterErr := &types.InvalidParameterException{}
	genericErr := errors.New("boom")

	if !ecrRegistryPolicyNotFound(registryPolicyErr) {
		t.Fatal("ecrRegistryPolicyNotFound() = false, want true")
	}
	if !ecrPullThroughCacheRuleNotFound(pullThroughErr) {
		t.Fatal("ecrPullThroughCacheRuleNotFound() = false, want true")
	}
	if !ecrTemplateNotFound(templateErr) {
		t.Fatal("ecrTemplateNotFound() = false, want true")
	}
	if !ecrInvalidParameter(invalidParameterErr) {
		t.Fatal("ecrInvalidParameter() = false, want true")
	}
	if ecrRegistryPolicyNotFound(genericErr) {
		t.Fatal("ecrRegistryPolicyNotFound() = true for generic error, want false")
	}
	if ecrPullThroughCacheRuleNotFound(genericErr) {
		t.Fatal("ecrPullThroughCacheRuleNotFound() = true for generic error, want false")
	}
	if ecrTemplateNotFound(genericErr) {
		t.Fatal("ecrTemplateNotFound() = true for generic error, want false")
	}
	if ecrInvalidParameter(genericErr) {
		t.Fatal("ecrInvalidParameter() = true for generic error, want false")
	}
	if ecrRegistryPolicyNotFound(nil) {
		t.Fatal("ecrRegistryPolicyNotFound() = true for nil error, want false")
	}
	if ecrPullThroughCacheRuleNotFound(nil) {
		t.Fatal("ecrPullThroughCacheRuleNotFound() = true for nil error, want false")
	}
	if ecrTemplateNotFound(nil) {
		t.Fatal("ecrTemplateNotFound() = true for nil error, want false")
	}
	if ecrInvalidParameter(nil) {
		t.Fatal("ecrInvalidParameter() = true for nil error, want false")
	}
}

func TestEcrOptionalRegistryResourcesContinueAfterError(t *testing.T) {
	g := EcrGenerator{}
	calledSecondLoader := false

	g.getOptionalRegistryResources(
		ecrOptionalResourceLoader{name: "denied", load: func() error { return errors.New("access denied") }},
		ecrOptionalResourceLoader{name: "next", load: func() error {
			calledSecondLoader = true
			return nil
		}},
	)

	if !calledSecondLoader {
		t.Fatal("getOptionalRegistryResources() stopped after optional loader error")
	}
}

func TestEcrPostConvertHookWrapsPolicyFields(t *testing.T) {
	registryPolicy := terraformutils.NewSimpleResource("123456789012", "registry-policy", "aws_ecr_registry_policy", "aws", ecrAllowEmptyValues)
	registryPolicy.Item = map[string]interface{}{"policy": "{\"Resource\":\"${aws:ecr}\"}"}

	template := terraformutils.NewSimpleResource("ROOT", "ROOT", "aws_ecr_repository_creation_template", "aws", ecrAllowEmptyValues)
	template.Item = map[string]interface{}{
		"repository_policy": "{\"Resource\":\"${aws:ecr}\"}",
		"lifecycle_policy":  "{\"rules\":[]}",
	}

	g := EcrGenerator{}
	g.Resources = []terraformutils.Resource{registryPolicy, template}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() returned error: %v", err)
	}

	wantRegistryPolicy := "<<POLICY\n{\"Resource\":\"$${aws:ecr}\"}\nPOLICY"
	if got := g.Resources[0].Item["policy"]; got != wantRegistryPolicy {
		t.Fatalf("registry policy = %q, want %q", got, wantRegistryPolicy)
	}

	wantRepositoryPolicy := "<<POLICY\n{\"Resource\":\"$${aws:ecr}\"}\nPOLICY"
	if got := g.Resources[1].Item["repository_policy"]; got != wantRepositoryPolicy {
		t.Fatalf("repository policy = %q, want %q", got, wantRepositoryPolicy)
	}

	wantLifecyclePolicy := "<<POLICY\n{\"rules\":[]}\nPOLICY"
	if got := g.Resources[1].Item["lifecycle_policy"]; got != wantLifecyclePolicy {
		t.Fatalf("lifecycle policy = %q, want %q", got, wantLifecyclePolicy)
	}
}

func TestEcrPublicRepositoryPolicyHelpers(t *testing.T) {
	if !ecrPublicRepositoryPolicyNotFound(&publictypes.RepositoryPolicyNotFoundException{}) {
		t.Fatal("ecrPublicRepositoryPolicyNotFound() = false, want true")
	}
	if !ecrPublicRepositoryPolicyNotFound(&publictypes.RepositoryNotFoundException{}) {
		t.Fatal("ecrPublicRepositoryPolicyNotFound() = false for missing repository, want true")
	}
	if ecrPublicRepositoryPolicyNotFound(errors.New("boom")) {
		t.Fatal("ecrPublicRepositoryPolicyNotFound() = true for generic error, want false")
	}
	if ecrPublicRepositoryPolicyNotFound(nil) {
		t.Fatal("ecrPublicRepositoryPolicyNotFound() = true for nil error, want false")
	}
}

func TestEcrPublicPostConvertHookWrapsPolicy(t *testing.T) {
	repositoryPolicy := terraformutils.NewSimpleResource("public-repo", "public-repo", "aws_ecrpublic_repository_policy", "aws", ecrPublicAllowEmptyValues)
	repositoryPolicy.Item = map[string]interface{}{"policy": "{\"Resource\":\"${aws:ecr}\"}"}

	g := EcrPublicGenerator{}
	g.Resources = []terraformutils.Resource{repositoryPolicy}

	if err := g.PostConvertHook(); err != nil {
		t.Fatalf("PostConvertHook() returned error: %v", err)
	}

	want := "<<POLICY\n{\"Resource\":\"$${aws:ecr}\"}\nPOLICY"
	if got := g.Resources[0].Item["policy"]; got != want {
		t.Fatalf("policy = %q, want %q", got, want)
	}
}
