// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"os"
	"testing"
)

func TestComputeDBRuntimeUnsupportedResourceEntries(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}
	var unsupported struct {
		Resources []struct {
			Resource      string   `json:"resource"`
			ServiceFamily string   `json:"service_family"`
			Reason        string   `json:"reason"`
			Evidence      string   `json:"evidence"`
			Status        string   `json:"status"`
			References    []string `json:"references"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(data, &unsupported); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}

	want := map[string]struct {
		serviceFamily string
		status        string
	}{
		"aws_db_instance_automated_backups_replication": {serviceFamily: "db", status: "unsupported"},
		"aws_db_snapshot_copy":                          {serviceFamily: "db", status: "unsupported"},
		"aws_docdb_cluster_snapshot":                    {serviceFamily: "docdb", status: "unsupported"},
		"aws_memorydb_multi_region_cluster":             {serviceFamily: "memorydb", status: "deferred"},
		"aws_memorydb_snapshot":                         {serviceFamily: "memorydb", status: "unsupported"},
		"aws_memorydb_user":                             {serviceFamily: "memorydb", status: "unsupported"},
		"aws_neptune_cluster_snapshot":                  {serviceFamily: "neptune", status: "unsupported"},
		"aws_rds_cluster_snapshot_copy":                 {serviceFamily: "rds", status: "unsupported"},
		"aws_rds_export_task":                           {serviceFamily: "rds", status: "unsupported"},
		"aws_rds_instance_state":                        {serviceFamily: "rds", status: "unsupported"},
		"aws_rds_reserved_instance":                     {serviceFamily: "rds", status: "unsupported"},
	}
	found := map[string]bool{}
	for _, entry := range unsupported.Resources {
		expected, ok := want[entry.Resource]
		if !ok {
			continue
		}
		found[entry.Resource] = true
		if entry.ServiceFamily != expected.serviceFamily {
			t.Fatalf("%s service family = %q, want %q", entry.Resource, entry.ServiceFamily, expected.serviceFamily)
		}
		if entry.Status != expected.status {
			t.Fatalf("%s status = %q, want %q", entry.Resource, entry.Status, expected.status)
		}
		if entry.Reason == "" || entry.Evidence == "" || len(entry.References) == 0 {
			t.Fatalf("%s unsupported entry is missing reason, evidence, or references", entry.Resource)
		}
	}
	for resource := range want {
		if !found[resource] {
			t.Fatalf("%s unsupported entry was not found", resource)
		}
	}
}
