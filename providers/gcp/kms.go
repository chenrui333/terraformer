// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/cloudkms/v1"

	"github.com/chenrui333/terraformer/terraformutils"
)

var kmsAllowEmptyValues = []string{""}

var kmsAdditionalFields = map[string]interface{}{}

type KmsGenerator struct {
	GCPService
}

func (g KmsGenerator) createKmsRingResources(ctx context.Context, keyRingList *cloudkms.ProjectsLocationsKeyRingsListCall, kmsService *cloudkms.Service) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := keyRingList.Pages(ctx, func(page *cloudkms.ListKeyRingsResponse) error {
		for _, obj := range page.KeyRings {
			tm := strings.Split(obj.Name, "/")
			ID := tm[1] + "/" + tm[3] + "/" + tm[5]
			resources = append(resources, terraformutils.NewResource(
				ID,
				tm[len(tm)-3]+"_"+tm[len(tm)-1],
				"google_kms_key_ring",
				g.ProviderName,
				map[string]string{
					"project":  g.GetArgs()["project"].(string),
					"location": tm[3],
					"name":     tm[5],
				},
				kmsAllowEmptyValues,
				kmsAdditionalFields,
			))
			keyResources, err := g.createKmsKeyResources(ctx, obj.Name, kmsService)
			if err != nil {
				return err
			}
			resources = append(resources, keyResources...)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list kms key rings: %w", err)
	}
	return resources, nil
}

func (g *KmsGenerator) createKmsKeyResources(ctx context.Context, keyRingName string, kmsService *cloudkms.Service) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	keyList := kmsService.Projects.Locations.KeyRings.CryptoKeys.List(keyRingName)
	if err := keyList.Pages(ctx, func(page *cloudkms.ListCryptoKeysResponse) error {
		for _, key := range page.CryptoKeys {
			tm := strings.Split(key.Name, "/")
			resources = append(resources, terraformutils.NewResource(
				key.Name,
				tm[1]+"_"+tm[3]+"_"+tm[5]+"_"+tm[7],
				"google_kms_crypto_key",
				g.ProviderName,
				map[string]string{
					"project": g.GetArgs()["project"].(string),
					"name":    key.Name,
				},
				kmsAllowEmptyValues,
				kmsAdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list kms crypto keys for %s: %w", keyRingName, err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
func (g *KmsGenerator) InitResources() error {
	ctx := context.Background()
	kmsService, err := cloudkms.NewService(ctx)
	if err != nil {
		return err
	}

	keyRingList := kmsService.Projects.Locations.KeyRings.List("projects/" + g.GetArgs()["project"].(string) + "/locations/global")

	resources, err := g.createKmsRingResources(ctx, keyRingList, kmsService)
	if err != nil {
		return err
	}
	g.Resources = resources
	return nil
}

func (g *KmsGenerator) PostConvertHook() error {
	for i, key := range g.Resources {
		if key.InstanceInfo.Type != "google_kms_crypto_key" {
			continue
		}
		for _, keyRing := range g.Resources {
			if keyRing.InstanceInfo.Type != "google_kms_key_ring" {
				continue
			}
			if key.Item["key_ring"] == keyRing.InstanceState.ID {
				g.Resources[i].Item["key_ring"] = "${google_kms_key_ring." + keyRing.ResourceName + ".self_link}"
			}
		}
	}
	return nil
}
