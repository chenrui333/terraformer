// SPDX-License-Identifier: Apache-2.0

package datadog

import (
	"context"
	"fmt"
	"strings"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/chenrui333/terraformer/terraformutils"
)

var (
	OrgConnectionAllowEmptyValues = []string{}
)

type OrgConnectionGenerator struct {
	DatadogService
}

func (g *OrgConnectionGenerator) createResource(conn datadogV2.OrgConnection) terraformutils.Resource {
	id := conn.GetId().String()
	resourceName := fmt.Sprintf("org_connection_%s", id)

	attrs := map[string]string{}

	rels := conn.GetRelationships()
	sinkOrg := rels.GetSinkOrg()
	if sinkOrg.Data != nil {
		sinkData := sinkOrg.GetData()
		if sinkID := (&sinkData).GetId(); sinkID != "" {
			attrs["sink_org_id"] = sinkID
		}
	}

	connAttrs := conn.GetAttributes()
	connTypes := (&connAttrs).GetConnectionTypes()
	typeStrs := make([]string, 0, len(connTypes))
	for _, ct := range connTypes {
		typeStrs = append(typeStrs, string(ct))
	}
	attrs["connection_types.#"] = fmt.Sprintf("%d", len(typeStrs))
	for i, t := range typeStrs {
		attrs[fmt.Sprintf("connection_types.%d", i)] = t
	}

	return terraformutils.NewResource(
		id,
		resourceName,
		"datadog_org_connection",
		"datadog",
		attrs,
		OrgConnectionAllowEmptyValues,
		map[string]interface{}{},
	)
}

func (g *OrgConnectionGenerator) InitResources() error {
	datadogClient := g.Args["datadogClient"].(*datadog.APIClient)
	auth := g.Args["auth"].(context.Context)
	api := datadogV2.NewOrgConnectionsApi(datadogClient)

	resources := []terraformutils.Resource{}
	var offset int64
	const limit int64 = 100

	for {
		opts := datadogV2.NewListOrgConnectionsOptionalParameters().WithLimit(limit).WithOffset(offset)
		resp, httpResp, err := api.ListOrgConnections(auth, *opts)
		if httpResp != nil && httpResp.Body != nil {
			_ = httpResp.Body.Close()
		}
		if err != nil {
			if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "404") {
				break
			}
			return err
		}

		data := resp.GetData()
		for _, conn := range data {
			if conn.GetId().String() == "00000000-0000-0000-0000-000000000000" {
				continue
			}
			resources = append(resources, g.createResource(conn))
		}

		if int64(len(data)) < limit {
			break
		}
		offset += limit
	}

	g.Resources = resources
	return nil
}
