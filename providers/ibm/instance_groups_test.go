// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	instanceGroupsPath        = "/instance_groups"
	instanceGroupManagerPath  = "/instance_groups/ig-1/managers/m-1"
	instanceGroupManager2Path = "/instance_groups/ig-1/managers/m-2"
	instanceGroupPolicyPath   = "/instance_groups/ig-1/managers/m-1/policies/p-1"
	instanceGroupsBody        = `{"instance_groups":[{"id":"ig-1","name":"group","managers":[{"id":"m-1"}]}]}`
	unnamedManagerBody        = `{"id":"m-1","policies":[{"id":"p-1"}]}`
	instanceGroupPolicyBody   = `{"id":"p-1"}`
	instanceGroupTestDeadline = 10 * time.Second
)

func jsonResponse(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

func instanceGroupRoutes(managerBody string) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		instanceGroupsPath:       jsonResponse(instanceGroupsBody),
		instanceGroupManagerPath: jsonResponse(managerBody),
		instanceGroupPolicyPath:  jsonResponse(instanceGroupPolicyBody),
	}
}

// newInstanceGroupTestSession serves routes; any other path fails with HTTP 500.
func newInstanceGroupTestSession(t *testing.T, routes map[string]http.HandlerFunc) *vpcv1.VpcV1 {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if route, ok := routes[r.URL.Path]; ok {
			route(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	t.Cleanup(server.Close)

	sess, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		URL:           server.URL,
		Authenticator: &core.NoAuthAuthenticator{},
	})
	if err != nil {
		t.Fatalf("creating VPC client: %v", err)
	}
	return sess
}

func runInstanceGroups(t *testing.T, g *InstanceGroupGenerator, sess *vpcv1.VpcV1) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- g.handleInstanceGroups(sess) }()
	select {
	case err := <-done:
		return err
	case <-time.After(instanceGroupTestDeadline):
		t.Fatal("instance group import did not return")
		return nil
	}
}

func resourcesByType(resources []terraformutils.Resource) map[string]terraformutils.Resource {
	byType := map[string]terraformutils.Resource{}
	for _, resource := range resources {
		byType[resource.InstanceInfo.Type] = resource
	}
	return byType
}

func TestInstanceGroupsPropagateAPIErrors(t *testing.T) {
	tests := map[string]struct {
		failPath string
		want     string
	}{
		"list instance groups": {failPath: instanceGroupsPath, want: "error fetching InstanceGroups"},
		"get manager":          {failPath: instanceGroupManagerPath, want: "error getting InstanceGroup Manager:"},
		"get manager policy":   {failPath: instanceGroupPolicyPath, want: "error getting InstanceGroup Manager Policy"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			routes := instanceGroupRoutes(unnamedManagerBody)
			delete(routes, tt.failPath)
			g := &InstanceGroupGenerator{}
			err := runInstanceGroups(t, g, newInstanceGroupTestSession(t, routes))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestInstanceGroupsManagerErrorWaitsForPolicyWorkers(t *testing.T) {
	routes := instanceGroupRoutes(unnamedManagerBody)
	routes[instanceGroupsPath] = jsonResponse(`{"instance_groups":[{"id":"ig-1","name":"group","managers":[{"id":"m-1"},{"id":"m-2"}]}]}`)
	secondManagerFailed := make(chan struct{})
	routes[instanceGroupManager2Path] = func(w http.ResponseWriter, _ *http.Request) {
		close(secondManagerFailed)
		w.WriteHeader(http.StatusInternalServerError)
	}
	// Answer m-1's policy only after m-2 has failed, so returning early would
	// leave this worker running.
	routes[instanceGroupPolicyPath] = func(w http.ResponseWriter, r *http.Request) {
		<-secondManagerFailed
		time.Sleep(100 * time.Millisecond)
		jsonResponse(instanceGroupPolicyBody)(w, r)
	}

	g := &InstanceGroupGenerator{}
	err := runInstanceGroups(t, g, newInstanceGroupTestSession(t, routes))
	if err == nil || !strings.Contains(err.Error(), "error getting InstanceGroup Manager:") {
		t.Fatalf("expected manager error, got %v", err)
	}
	if _, ok := resourcesByType(g.Resources)["ibm_is_instance_group_manager_policy"]; !ok {
		t.Fatalf("expected the m-1 policy worker to finish before returning, got %d resources", len(g.Resources))
	}
}

func TestInstanceGroupsChildNamesAndDependencies(t *testing.T) {
	tests := map[string]struct {
		managerBody string
		managerName string
	}{
		"unnamed manager": {managerBody: unnamedManagerBody, managerName: "id_m-1"},
		"named manager":   {managerBody: `{"id":"m-1","name":"web-manager","policies":[{"id":"p-1"}]}`, managerName: "web-manager"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			g := &InstanceGroupGenerator{}
			if err := runInstanceGroups(t, g, newInstanceGroupTestSession(t, instanceGroupRoutes(tt.managerBody))); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			byType := resourcesByType(g.Resources)
			if len(g.Resources) != 3 {
				t.Fatalf("expected 3 resources, got %d", len(g.Resources))
			}
			wantNames := map[string]string{
				"ibm_is_instance_group":                "group",
				"ibm_is_instance_group_manager":        tt.managerName,
				"ibm_is_instance_group_manager_policy": "id_p-1",
			}
			for resourceType, name := range wantNames {
				if got := byType[resourceType].ResourceName; got != terraformutils.TfSanitize(name) {
					t.Errorf("%s: expected resource name %q, got %q", resourceType, terraformutils.TfSanitize(name), got)
				}
			}

			wantDependsOn := []string{
				"ibm_is_instance_group." + byType["ibm_is_instance_group"].ResourceName,
				"ibm_is_instance_group_manager." + byType["ibm_is_instance_group_manager"].ResourceName,
			}
			gotDependsOn := byType["ibm_is_instance_group_manager_policy"].AdditionalFields["depends_on"]
			if !reflect.DeepEqual(gotDependsOn, wantDependsOn) {
				t.Fatalf("expected policy depends_on %v, got %v", wantDependsOn, gotDependsOn)
			}
		})
	}
}
