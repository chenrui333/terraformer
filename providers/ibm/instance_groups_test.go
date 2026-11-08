// SPDX-License-Identifier: Apache-2.0

package ibm

import (
	"net/http"
	"net/http/httptest"
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
	instanceGroupPolicyPath   = "/instance_groups/ig-1/managers/m-1/policies/p-1"
	instanceGroupsBody        = `{"instance_groups":[{"id":"ig-1","name":"group","managers":[{"id":"m-1"}]}]}`
	instanceGroupManagerBody  = `{"id":"m-1","name":"","policies":[{"id":"p-1"}]}`
	instanceGroupPolicyBody   = `{"id":"p-1","name":""}`
	instanceGroupTestDeadline = 10 * time.Second
)

func newInstanceGroupTestSession(t *testing.T, failPath string) *vpcv1.VpcV1 {
	t.Helper()
	bodies := map[string]string{
		instanceGroupsPath:       instanceGroupsBody,
		instanceGroupManagerPath: instanceGroupManagerBody,
		instanceGroupPolicyPath:  instanceGroupPolicyBody,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, ok := bodies[r.URL.Path]
		if !ok || r.URL.Path == failPath {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
			return
		}
		_, _ = w.Write([]byte(body))
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
			g := &InstanceGroupGenerator{}
			err := runInstanceGroups(t, g, newInstanceGroupTestSession(t, tt.failPath))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestInstanceGroupsUnnamedChildrenUseIDs(t *testing.T) {
	g := &InstanceGroupGenerator{}
	if err := runInstanceGroups(t, g, newInstanceGroupTestSession(t, "")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := map[string]string{}
	for _, resource := range g.Resources {
		names[resource.InstanceInfo.Type] = resource.ResourceName
	}
	want := map[string]string{
		"ibm_is_instance_group":                "group",
		"ibm_is_instance_group_manager":        "m-1",
		"ibm_is_instance_group_manager_policy": "p-1",
	}
	for resourceType, name := range want {
		if got := names[resourceType]; got != terraformutils.TfSanitize(name) {
			t.Errorf("%s: expected resource name %q, got %q", resourceType, terraformutils.TfSanitize(name), got)
		}
	}
}
