// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNetworkTGWUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported map[string]interface{}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	entries, ok := unsupported["resources"].([]interface{})
	if !ok {
		t.Fatal("unsupported resources file is missing resources list")
	}

	want := map[string]struct {
		serviceFamily string
		status        string
	}{
		"aws_dx_connection_confirmation":                              {"dx", "unsupported"},
		"aws_dx_macsec_key_association":                               {"dx", "unsupported"},
		"aws_ec2_transit_gateway_default_route_table_association":     {"transit_gateway", "deferred"},
		"aws_ec2_transit_gateway_multicast_domain_association":        {"transit_gateway", "not-importable"},
		"aws_ec2_transit_gateway_multicast_group_member":              {"transit_gateway", "not-importable"},
		"aws_ec2_transit_gateway_vpc_attachment_accepter":             {"transit_gateway", "unsupported"},
		"aws_networkmanager_attachment_accepter":                      {"networkmanager", "unsupported"},
		"aws_networkmanager_attachment_routing_policy_label":          {"networkmanager", "deferred"},
		"aws_networkmanager_connect_attachment":                       {"networkmanager", "deferred"},
		"aws_networkmanager_connect_peer":                             {"networkmanager", "deferred"},
		"aws_networkmanager_core_network":                             {"networkmanager", "deferred"},
		"aws_networkmanager_core_network_policy_attachment":           {"networkmanager", "deferred"},
		"aws_networkmanager_customer_gateway_association":             {"networkmanager", "deferred"},
		"aws_networkmanager_dx_gateway_attachment":                    {"networkmanager", "deferred"},
		"aws_networkmanager_link_association":                         {"networkmanager", "deferred"},
		"aws_networkmanager_prefix_list_association":                  {"networkmanager", "deferred"},
		"aws_networkmanager_site_to_site_vpn_attachment":              {"networkmanager", "deferred"},
		"aws_networkmanager_transit_gateway_connect_peer_association": {"networkmanager", "deferred"},
		"aws_networkmanager_transit_gateway_peering":                  {"networkmanager", "deferred"},
		"aws_networkmanager_transit_gateway_registration":             {"networkmanager", "deferred"},
		"aws_networkmanager_transit_gateway_route_table_attachment":   {"networkmanager", "deferred"},
		"aws_networkmanager_vpc_attachment":                           {"networkmanager", "deferred"},
		"aws_route53_records_exclusive":                               {"route53", "deferred"},
		"aws_route53_vpc_association_authorization":                   {"route53", "unsupported"},
	}

	found := map[string]bool{}
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			t.Fatalf("unsupported resource entry has unexpected type %T", rawEntry)
		}
		resource, _ := entry["resource"].(string)
		expected, ok := want[resource]
		if !ok {
			continue
		}
		found[resource] = true
		if serviceFamily, _ := entry["service_family"].(string); serviceFamily != expected.serviceFamily {
			t.Fatalf("%s service family = %q, want %q", resource, serviceFamily, expected.serviceFamily)
		}
		if status, _ := entry["status"].(string); status != expected.status {
			t.Fatalf("%s status = %q, want %q", resource, status, expected.status)
		}
		reason, _ := entry["reason"].(string)
		evidence, _ := entry["evidence"].(string)
		references, _ := entry["references"].([]interface{})
		if reason == "" || evidence == "" || len(references) == 0 {
			t.Fatalf("%s entry is missing reason, evidence, or references", resource)
		}
	}
	for resource := range want {
		if !found[resource] {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
}
