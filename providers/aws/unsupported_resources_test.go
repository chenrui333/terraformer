// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const awsUnsupportedIssue = "https://github.com/chenrui333/terraformer/issues/338"

type awsUnsupportedResourcesFile struct {
	Version   int                                    `json:"version"`
	Resources []awsUnsupportedResourceMetadataRecord `json:"resources"`
}

type awsUnsupportedResourceMetadataRecord struct {
	Resource      string   `json:"resource"`
	ServiceFamily string   `json:"service_family"`
	Reason        string   `json:"reason"`
	Evidence      string   `json:"evidence"`
	Status        string   `json:"status"`
	References    []string `json:"references"`
}

func TestAWSUnsupportedResourcesMetadata(t *testing.T) {
	data, err := os.ReadFile("unsupported_resources.json")
	if err != nil {
		t.Fatalf("read unsupported resources: %v", err)
	}

	var metadata awsUnsupportedResourcesFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode unsupported resources: %v", err)
	}
	if err := validateAWSUnsupportedResourcesMetadata(metadata); err != nil {
		t.Fatal(err)
	}
}

func TestAWSUnsupportedResourcesMetadataDoesNotRedefineSharedStatuses(t *testing.T) {
	metadata := awsUnsupportedResourcesFile{
		Version: 1,
		Resources: []awsUnsupportedResourceMetadataRecord{
			{
				Resource:      "aws_example_action",
				ServiceFamily: "example",
				Reason:        "The resource represents an action.",
				Evidence:      "The provider invokes an operation instead of managing durable configuration.",
				Status:        "action-style",
				References:    []string{awsUnsupportedIssue},
			},
		},
	}
	if err := validateAWSUnsupportedResourcesMetadata(metadata); err != nil {
		t.Fatalf("AWS-local validation rejected shared canonical status: %v", err)
	}
}

func validateAWSUnsupportedResourcesMetadata(metadata awsUnsupportedResourcesFile) error {
	if metadata.Version != 1 {
		return fmt.Errorf("unsupported resources version = %d, want 1", metadata.Version)
	}
	if len(metadata.Resources) == 0 {
		return fmt.Errorf("unsupported resources file is missing resources list")
	}

	seen := map[string]bool{}
	previousResource := ""
	for _, resource := range metadata.Resources {
		name := strings.TrimSpace(resource.Resource)
		if name == "" {
			return fmt.Errorf("unsupported resource entry is missing resource")
		}
		if !strings.HasPrefix(name, "aws_") {
			return fmt.Errorf("unsupported resource %q does not use aws_ prefix", name)
		}
		if seen[name] {
			return fmt.Errorf("unsupported resource %q is duplicated", name)
		}
		seen[name] = true
		if previousResource != "" && previousResource > name {
			return fmt.Errorf("unsupported resources are not sorted by resource: %q before %q", previousResource, name)
		}
		previousResource = name

		if strings.TrimSpace(resource.ServiceFamily) == "" {
			return fmt.Errorf("unsupported resource %q is missing service_family", name)
		}
		if strings.TrimSpace(resource.Reason) == "" {
			return fmt.Errorf("unsupported resource %q is missing reason", name)
		}
		if strings.TrimSpace(resource.Evidence) == "" {
			return fmt.Errorf("unsupported resource %q is missing evidence", name)
		}
		if !hasUnsupportedResourceReference(resource.References, awsUnsupportedIssue) {
			return fmt.Errorf("unsupported resource %q is missing issue #338 reference", name)
		}
		for _, reference := range resource.References {
			if strings.TrimSpace(reference) == "" {
				return fmt.Errorf("unsupported resource %q has an empty reference", name)
			}
		}
	}
	return nil
}

func hasUnsupportedResourceReference(references []string, expected string) bool {
	for _, reference := range references {
		if reference == expected {
			return true
		}
	}
	return false
}
