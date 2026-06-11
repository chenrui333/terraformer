// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	cf "github.com/chenrui333/terraformer/providers/cloudflare/internal/cloudflarev7"
	"github.com/chenrui333/terraformer/terraformutils"
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

func listAllListItems(ctx context.Context, api *cf.API, account *cf.ResourceContainer, listID string) ([]cf.ListItem, error) {
	items := []cf.ListItem{}
	cursor := ""
	for {
		values := url.Values{}
		values.Set("per_page", strconv.Itoa(cloudflarePageSize))
		if cursor != "" {
			values.Set("cursor", cursor)
		}
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/rules/lists/%s/items?%s", account.Identifier, listID, values.Encode()),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageItems []cf.ListItem
		if err := json.Unmarshal(response.Result, &pageItems); err != nil {
			return nil, err
		}
		items = append(items, pageItems...)
		if response.ResultInfo == nil || response.ResultInfo.Cursors.After == "" {
			break
		}
		cursor = response.ResultInfo.Cursors.After
	}
	return items, nil
}

func listAttributes(ctx context.Context, api *cf.API, account *cf.ResourceContainer, list cf.List) (map[string]string, error) {
	attributes := map[string]string{"account_id": account.Identifier}
	items, err := listAllListItems(ctx, api, account, list.ID)
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

func listAllLists(ctx context.Context, api *cf.API, account *cf.ResourceContainer) ([]cf.List, error) {
	lists := []cf.List{}
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/rules/lists?%s", account.Identifier, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageLists []cf.List
		if err := json.Unmarshal(response.Result, &pageLists); err != nil {
			return nil, err
		}
		lists = append(lists, pageLists...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return lists, nil
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
	lists, err := listAllLists(ctx, api, account)
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
