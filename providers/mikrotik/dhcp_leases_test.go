// SPDX-License-Identifier: Apache-2.0

package mikrotik

import (
	"testing"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/ddelnano/terraform-provider-mikrotik/client"
)

func TestDhcpLeaseGeneratorCreateResources(t *testing.T) {
	generator := DhcpLeaseGenerator{}

	resources := generator.createResources([]client.DhcpLease{
		{Id: "*1", Hostname: "laptop"},
		{Id: "*2"},
	})
	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want 2", len(resources))
	}

	if got, want := resources[0].InstanceState.ID, "*1"; got != want {
		t.Fatalf("first resource ID = %q, want %q", got, want)
	}
	if got, want := resources[0].ResourceName, terraformutils.TfSanitize("laptop-*1"); got != want {
		t.Fatalf("first resource name = %q, want %q", got, want)
	}
	if got, want := resources[0].InstanceInfo.Type, "mikrotik_dhcp_lease"; got != want {
		t.Fatalf("first resource type = %q, want %q", got, want)
	}

	if got, want := resources[1].InstanceState.ID, "*2"; got != want {
		t.Fatalf("second resource ID = %q, want %q", got, want)
	}
	if got, want := resources[1].ResourceName, terraformutils.TfSanitize("*2"); got != want {
		t.Fatalf("second resource name = %q, want %q", got, want)
	}
}
