// SPDX-License-Identifier: Apache-2.0

package cloudflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	cf "github.com/cloudflare/cloudflare-go"
)

type WorkersGenerator struct {
	CloudflareService
}

func (g *WorkersGenerator) InitResources() error {
	ctx := context.Background()
	api, err := g.initializeAPI()
	if err != nil {
		return err
	}
	zones, err := cloudflareZones(ctx, api)
	if err != nil {
		return err
	}
	for _, zone := range zones {
		routes, err := listWorkerRoutes(ctx, api, zone.ID)
		if err != nil {
			return err
		}
		for _, route := range routes {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				route.ID,
				cloudflareResourceName(zone.Name, route.Pattern, route.ID),
				"cloudflare_workers_route",
				"cloudflare",
				map[string]string{"zone_id": zone.ID},
				[]string{},
				map[string]interface{}{},
			))
		}
	}
	accountID := g.accountID()
	if accountID == "" {
		return nil
	}
	account := cf.AccountIdentifier(accountID)
	if err := g.appendWorkerCustomDomainResources(ctx, api, accountID); err != nil {
		return err
	}
	if err := g.appendWorkerCronTriggerResources(ctx, api, account); err != nil {
		return err
	}
	if err := g.appendWorkersForPlatformsDispatchNamespaceResources(ctx, api, accountID); err != nil {
		return err
	}
	return nil
}

func listWorkerRoutes(ctx context.Context, api *cf.API, zoneID string) ([]cf.WorkerRoute, error) {
	var routes []cf.WorkerRoute
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/zones/%s/workers/routes?%s", zoneID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageRoutes []cf.WorkerRoute
		if err := json.Unmarshal(response.Result, &pageRoutes); err != nil {
			return nil, err
		}
		routes = append(routes, pageRoutes...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return routes, nil
}

func listWorkerCustomDomains(ctx context.Context, api *cf.API, accountID string) ([]cf.WorkersDomain, error) {
	var domains []cf.WorkersDomain
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/workers/domains?%s", accountID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageDomains []cf.WorkersDomain
		if err := json.Unmarshal(response.Result, &pageDomains); err != nil {
			return nil, err
		}
		domains = append(domains, pageDomains...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return domains, nil
}

func listWorkerScripts(ctx context.Context, api *cf.API, accountID string) ([]cf.WorkerMetaData, error) {
	var scripts []cf.WorkerMetaData
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/workers/scripts?%s", accountID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageScripts []cf.WorkerMetaData
		if err := json.Unmarshal(response.Result, &pageScripts); err != nil {
			return nil, err
		}
		scripts = append(scripts, pageScripts...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return scripts, nil
}

func listWorkersForPlatformsDispatchNamespaces(
	ctx context.Context,
	api *cf.API,
	accountID string,
) ([]cf.WorkersForPlatformsDispatchNamespace, error) {
	var namespaces []cf.WorkersForPlatformsDispatchNamespace
	page, cursor := 1, ""
	for {
		response, err := api.Raw(
			ctx,
			http.MethodGet,
			fmt.Sprintf("/accounts/%s/workers/dispatch/namespaces?%s", accountID, cloudflarePaginationQuery(page, cursor)),
			nil,
			nil,
		)
		if err != nil {
			return nil, err
		}
		var pageNamespaces []cf.WorkersForPlatformsDispatchNamespace
		if err := json.Unmarshal(response.Result, &pageNamespaces); err != nil {
			return nil, err
		}
		namespaces = append(namespaces, pageNamespaces...)
		if !cloudflareAdvancePagination(response.ResultInfo, &page, &cursor) {
			break
		}
	}
	return namespaces, nil
}

func workerCustomDomainAttributes(accountID string, domain cf.WorkersDomain) map[string]string {
	attributes := map[string]string{
		"account_id": accountID,
		"hostname":   domain.Hostname,
		"service":    domain.Service,
	}
	if domain.Environment != "" {
		attributes["environment"] = domain.Environment
	}
	if domain.ZoneID != "" {
		attributes["zone_id"] = domain.ZoneID
	}
	if domain.ZoneName != "" {
		attributes["zone_name"] = domain.ZoneName
	}
	return attributes
}

func workerCronTriggerAttributes(
	accountID string,
	scriptName string,
	schedules []cf.WorkerCronTrigger,
) map[string]string {
	attributes := map[string]string{
		"account_id":  accountID,
		"script_name": scriptName,
		"schedules.#": strconv.Itoa(len(schedules)),
	}
	for index, schedule := range schedules {
		attributes[fmt.Sprintf("schedules.%d.cron", index)] = schedule.Cron
	}
	return attributes
}

func (g *WorkersGenerator) appendWorkerCustomDomainResources(ctx context.Context, api *cf.API, accountID string) error {
	domains, err := listWorkerCustomDomains(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, domain := range domains {
		if domain.ID == "" {
			continue
		}
		resource := terraformutils.NewResource(
			domain.ID,
			cloudflareResourceName(accountID, domain.Hostname, domain.ID),
			"cloudflare_workers_custom_domain",
			"cloudflare",
			workerCustomDomainAttributes(accountID, domain),
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+domain.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *WorkersGenerator) appendWorkerCronTriggerResources(
	ctx context.Context,
	api *cf.API,
	account *cf.ResourceContainer,
) error {
	scripts, err := listWorkerScripts(ctx, api, account.Identifier)
	if err != nil {
		return err
	}
	for _, script := range scripts {
		if script.ID == "" {
			continue
		}
		schedules, err := api.ListWorkerCronTriggers(ctx, account, cf.ListWorkerCronTriggersParams{ScriptName: script.ID})
		if err != nil {
			if cloudflareNotFoundError(err) {
				continue
			}
			return err
		}
		if len(schedules) == 0 {
			continue
		}
		resource := terraformutils.NewResource(
			script.ID,
			cloudflareResourceName(account.Identifier, script.ID, "cron_trigger"),
			"cloudflare_workers_cron_trigger",
			"cloudflare",
			workerCronTriggerAttributes(account.Identifier, script.ID, schedules),
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, account.Identifier+"/"+script.ID)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}

func (g *WorkersGenerator) appendWorkersForPlatformsDispatchNamespaceResources(
	ctx context.Context,
	api *cf.API,
	accountID string,
) error {
	namespaces, err := listWorkersForPlatformsDispatchNamespaces(ctx, api, accountID)
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		if namespace.NamespaceName == "" {
			continue
		}
		resource := terraformutils.NewResource(
			namespace.NamespaceName,
			cloudflareResourceName(accountID, namespace.NamespaceName),
			"cloudflare_workers_for_platforms_dispatch_namespace",
			"cloudflare",
			map[string]string{
				"account_id":     accountID,
				"name":           namespace.NamespaceName,
				"namespace_name": namespace.NamespaceName,
			},
			[]string{},
			map[string]interface{}{},
		)
		setCloudflareImportID(&resource, accountID+"/"+namespace.NamespaceName)
		g.Resources = append(g.Resources, resource)
	}
	return nil
}
