// SPDX-License-Identifier: Apache-2.0

package fastly

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/fastly/go-fastly/v15/fastly"
)

const (
	// ServiceTypeVCL is the type for VCL services.
	ServiceTypeVCL = "vcl"
	// ServiceTypeWasm is the type for Wasm services.
	ServiceTypeWasm = "wasm"
)

type ServiceV1Generator struct {
	FastlyService
}

func (g *ServiceV1Generator) loadServices(client *fastly.Client) ([]*fastly.Service, error) {
	ctx := context.Background()
	services, err := client.ListServices(ctx, &fastly.ListServicesInput{})
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		serviceID := fastlyStringValue(service.ServiceID)
		if serviceID == "" {
			continue
		}
		switch fastlyStringValue(service.Type) {
		case ServiceTypeVCL:
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				serviceID,
				serviceID,
				"fastly_service_v1",
				"fastly",
				[]string{}))
		case ServiceTypeWasm:
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				serviceID,
				serviceID,
				"fastly_service_compute",
				"fastly",
				[]string{}))
		}
	}
	return services, nil
}

func (g *ServiceV1Generator) loadDictionaryItems(client *fastly.Client, serviceID string) error {
	ctx := context.Background()
	latest, err := client.LatestVersion(ctx, &fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	latestVersion := 0
	if latest != nil {
		latestVersion = fastlyIntValue(latest.Number)
	}
	if latestVersion == 0 {
		return nil
	}
	dictionaries, err := client.ListDictionaries(ctx, &fastly.ListDictionariesInput{
		ServiceID:      serviceID,
		ServiceVersion: latestVersion,
	})
	if err != nil {
		return err
	}
	for _, dictionary := range dictionaries {
		dictionaryID := fastlyStringValue(dictionary.DictionaryID)
		if dictionaryID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			dictionaryID,
			dictionaryID,
			"fastly_service_dictionary_items_v1",
			"fastly",
			map[string]string{
				"service_id":    serviceID,
				"dictionary_id": dictionaryID,
			},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *ServiceV1Generator) loadACLEntries(client *fastly.Client, serviceID string) error {
	ctx := context.Background()
	latest, err := client.LatestVersion(ctx, &fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	latestVersion := 0
	if latest != nil {
		latestVersion = fastlyIntValue(latest.Number)
	}
	if latestVersion == 0 {
		return nil
	}
	acls, err := client.ListACLs(ctx, &fastly.ListACLsInput{
		ServiceID:      serviceID,
		ServiceVersion: latestVersion,
	})
	if err != nil {
		return err
	}
	for _, acl := range acls {
		aclID := fastlyStringValue(acl.ACLID)
		if aclID == "" {
			continue
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			aclID,
			aclID,
			"fastly_service_acl_entries_v1",
			"fastly",
			map[string]string{
				"service_id": serviceID,
				"acl_id":     aclID,
			},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *ServiceV1Generator) loadDynamicSnippetContent(client *fastly.Client, serviceID string) error {
	ctx := context.Background()
	latest, err := client.LatestVersion(ctx, &fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	latestVersion := 0
	if latest != nil {
		latestVersion = fastlyIntValue(latest.Number)
	}
	if latestVersion == 0 {
		return nil
	}
	snippets, err := client.ListSnippets(ctx, &fastly.ListSnippetsInput{
		ServiceID:      serviceID,
		ServiceVersion: latestVersion,
	})
	if err != nil {
		return err
	}
	for _, snippet := range snippets {
		snippetID := fastlyStringValue(snippet.SnippetID)
		if snippetID == "" {
			continue
		}
		// check if dynamic
		if fastlyIntValue(snippet.Dynamic) == 1 {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				snippetID,
				snippetID,
				"fastly_service_dynamic_snippet_content_v1",
				"fastly",
				map[string]string{
					"service_id": serviceID,
					"snippet_id": snippetID,
				},
				[]string{},
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *ServiceV1Generator) InitResources() error {
	client, err := fastly.NewClient(g.Args["api_key"].(string))
	if err != nil {
		return err
	}
	services, err := g.loadServices(client)
	if err != nil {
		return err
	}
	for _, service := range services {
		serviceID := fastlyStringValue(service.ServiceID)
		if serviceID == "" {
			continue
		}
		err := g.loadDictionaryItems(client, serviceID)
		if err != nil {
			return err
		}
		err = g.loadACLEntries(client, serviceID)
		if err != nil {
			return err
		}
		err = g.loadDynamicSnippetContent(client, serviceID)
		if err != nil {
			return err
		}
	}
	return nil
}
