// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"fmt"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type ListsGenerator struct {
	CloudflareService
}

func addListItemAttributes(attributes map[string]string, index int, item cf.ListItem) {
	prefix := fmt.Sprintf("items.%d", index)
	if item.IP != nil {
		attributes[prefix+".ip"] = *item.IP
	}
	if item.ASN != nil {
		attributes[prefix+".asn"] = strconv.FormatUint(uint64(*item.ASN), 10)
	}
	if item.Comment != "" {
		attributes[prefix+".comment"] = item.Comment
	}
	if item.Hostname != nil {
		attributes[prefix+".hostname.url_hostname"] = item.Hostname.UrlHostname
	}
	if item.Redirect != nil {
		attributes[prefix+".redirect.source_url"] = item.Redirect.SourceUrl
		attributes[prefix+".redirect.target_url"] = item.Redirect.TargetUrl
		if item.Redirect.IncludeSubdomains != nil {
			attributes[prefix+".redirect.include_subdomains"] = strconv.FormatBool(*item.Redirect.IncludeSubdomains)
		}
		if item.Redirect.PreservePathSuffix != nil {
			attributes[prefix+".redirect.preserve_path_suffix"] = strconv.FormatBool(*item.Redirect.PreservePathSuffix)
		}
		if item.Redirect.PreserveQueryString != nil {
			attributes[prefix+".redirect.preserve_query_string"] = strconv.FormatBool(*item.Redirect.PreserveQueryString)
		}
		if item.Redirect.StatusCode != nil {
			attributes[prefix+".redirect.status_code"] = strconv.Itoa(*item.Redirect.StatusCode)
		}
		if item.Redirect.SubpathMatching != nil {
			attributes[prefix+".redirect.subpath_matching"] = strconv.FormatBool(*item.Redirect.SubpathMatching)
		}
	}
}

func listAttributes(ctx context.Context, api *cf.API, account *cf.ResourceContainer, list cf.List) (map[string]string, error) {
	attributes := map[string]string{"account_id": account.Identifier}
	items, err := api.ListListItems(ctx, account, cf.ListListItemsParams{ID: list.ID, PerPage: cloudflarePageSize})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return attributes, nil
	}
	attributes["items.#"] = strconv.Itoa(len(items))
	for index, item := range items {
		addListItemAttributes(attributes, index, item)
	}
	return attributes, nil
}

func (g *ListsGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	account, err := g.accountResourceContainer()
	if err != nil {
		return err
	}
	lists, err := api.ListLists(ctx, account, cf.ListListsParams{})
	if err != nil {
		return err
	}
	for _, list := range lists {
		attributes, err := listAttributes(ctx, api, account, list)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, terraformutils.NewResource(
			list.ID,
			cloudflareResourceName(account.Identifier, list.Name, list.ID),
			"cloudflare_list",
			"cloudflare",
			attributes,
			[]string{},
			map[string]interface{}{},
		))
	}
	return nil
}
