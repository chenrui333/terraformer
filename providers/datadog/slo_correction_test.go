// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"testing"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
)

func TestSLOCorrectionCreateResource(t *testing.T) {
	sloCorrection := datadogV1.NewSLOCorrectionWithDefaults()
	sloCorrection.SetId("correction-id")

	generator := &SLOCorrectionGenerator{}
	resource, err := generator.createResource(*sloCorrection)
	if err != nil {
		t.Fatalf("createResource returned error: %v", err)
	}

	if resource.InstanceState.ID != "correction-id" {
		t.Fatalf("resource ID = %q, want %q", resource.InstanceState.ID, "correction-id")
	}
	if resource.ResourceName != "tfer--slo_correction_correction-id" {
		t.Fatalf("resource name = %q, want %q", resource.ResourceName, "tfer--slo_correction_correction-id")
	}
	if resource.InstanceInfo.Type != "datadog_slo_correction" {
		t.Fatalf("resource type = %q, want %q", resource.InstanceInfo.Type, "datadog_slo_correction")
	}
}

func TestSLOCorrectionCreateResourceMissingID(t *testing.T) {
	generator := &SLOCorrectionGenerator{}
	_, err := generator.createResource(datadogV1.SLOCorrection{})
	if err == nil {
		t.Fatal("createResource returned nil error, want missing id error")
	}
}

func TestSLOCorrectionCreateResources(t *testing.T) {
	firstCorrection := datadogV1.NewSLOCorrectionWithDefaults()
	firstCorrection.SetId("correction-1")
	secondCorrection := datadogV1.NewSLOCorrectionWithDefaults()
	secondCorrection.SetId("correction-2")

	generator := &SLOCorrectionGenerator{}
	resources, err := generator.createResources([]datadogV1.SLOCorrection{
		*firstCorrection,
		*secondCorrection,
	})
	if err != nil {
		t.Fatalf("createResources returned error: %v", err)
	}

	if len(resources) != 2 {
		t.Fatalf("resource count = %d, want %d", len(resources), 2)
	}
	if resources[0].ResourceName == resources[1].ResourceName {
		t.Fatalf("resource names should be unique, got %q", resources[0].ResourceName)
	}
}

func TestSLOCorrectionsFromRawData(t *testing.T) {
	sloCorrections := sloCorrectionsFromRawData([]interface{}{
		map[string]interface{}{
			"id":   "correction-1",
			"type": "correction",
		},
		map[string]interface{}{
			"id": "correction-2",
		},
		map[string]interface{}{
			"id":   "ignored-type",
			"type": "unknown",
		},
		map[string]interface{}{
			"type": "correction",
		},
	})

	if len(sloCorrections) != 2 {
		t.Fatalf("correction count = %d, want %d", len(sloCorrections), 2)
	}
	if sloCorrections[0].GetId() != "correction-1" {
		t.Fatalf("first correction ID = %q, want %q", sloCorrections[0].GetId(), "correction-1")
	}
	if sloCorrections[1].GetId() != "correction-2" {
		t.Fatalf("second correction ID = %q, want %q", sloCorrections[1].GetId(), "correction-2")
	}
}

func TestSLOCorrectionFromRawDataRejectsNonObjects(t *testing.T) {
	if _, ok := sloCorrectionFromRawData("correction-id"); ok {
		t.Fatal("sloCorrectionFromRawData accepted non-object raw data")
	}
}
