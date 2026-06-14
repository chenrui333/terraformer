// SPDX-License-Identifier: Apache-2.0

package vultr

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/chenrui333/terraformer/terraformutils"
	"github.com/vultr/govultr/v3"
)

type ServerGenerator struct {
	VultrService
}

type vultrInstanceAttachments struct {
	vpcIDs  []string
	vpc2IDs []string
}

func (g ServerGenerator) createResources(serverList []govultr.Instance, attachments map[string]vultrInstanceAttachments) []terraformutils.Resource {
	var resources []terraformutils.Resource
	for _, server := range serverList {
		resources = append(resources, terraformutils.NewResource(
			server.ID,
			server.ID,
			"vultr_instance",
			"vultr",
			vultrInstanceAttachmentAttributes(attachments[server.ID]),
			[]string{},
			map[string]interface{}{}))
	}
	return resources
}

func vultrInstanceAttachmentAttributes(attachments vultrInstanceAttachments) map[string]string {
	attributes := map[string]string{}
	addVultrStringSetAttributes(attributes, "vpc_ids", attachments.vpcIDs)
	addVultrStringSetAttributes(attributes, "vpc2_ids", attachments.vpc2IDs)
	return attributes
}

func addVultrStringSetAttributes(attributes map[string]string, field string, values []string) {
	uniqueValues := uniqueVultrStrings(values)
	if len(uniqueValues) == 0 {
		return
	}
	attributes[field+".#"] = strconv.Itoa(len(uniqueValues))
	for _, value := range uniqueValues {
		attributes[fmt.Sprintf("%s.%d", field, terraformutils.HashString(value))] = value
	}
}

func uniqueVultrStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	uniqueValues := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		uniqueValues = append(uniqueValues, value)
	}
	return uniqueValues
}

func (g *ServerGenerator) loadInstanceAttachments(client *govultr.Client, instances []govultr.Instance) (map[string]vultrInstanceAttachments, error) {
	attachments := map[string]vultrInstanceAttachments{}
	for _, instance := range instances {
		vpcIDs, err := listVultrInstanceVPCIDs(client, instance.ID)
		if err != nil {
			return nil, err
		}
		vpc2IDs, err := listVultrInstanceVPC2IDs(client, instance.ID)
		if err != nil {
			return nil, err
		}
		attachments[instance.ID] = vultrInstanceAttachments{
			vpcIDs:  vpcIDs,
			vpc2IDs: vpc2IDs,
		}
	}
	return attachments, nil
}

func listVultrInstanceVPCIDs(client *govultr.Client, instanceID string) ([]string, error) {
	vpcs, err := listAllVultrResources(context.Background(), func(ctx context.Context, opt *govultr.ListOptions) ([]govultr.VPCInfo, *govultr.Meta, *http.Response, error) {
		return client.Instance.ListVPCInfo(ctx, instanceID, opt)
	})
	if err != nil {
		return nil, fmt.Errorf("list vultr VPC attachments for instance %q: %w", instanceID, err)
	}
	ids := make([]string, 0, len(vpcs))
	for _, vpc := range vpcs {
		ids = append(ids, vpc.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

func listVultrInstanceVPC2IDs(client *govultr.Client, instanceID string) ([]string, error) {
	//nolint:staticcheck // Existing instances can still have deprecated VPC2 attachments that must be preserved.
	vpcs, err := listAllVultrResources(context.Background(), func(ctx context.Context, opt *govultr.ListOptions) ([]govultr.VPC2Info, *govultr.Meta, *http.Response, error) {
		//nolint:staticcheck // Existing instances can still have deprecated VPC2 attachments that must be preserved.
		return client.Instance.ListVPC2Info(ctx, instanceID, opt)
	})
	if err != nil {
		return nil, fmt.Errorf("list vultr VPC2 attachments for instance %q: %w", instanceID, err)
	}
	ids := make([]string, 0, len(vpcs))
	for _, vpc := range vpcs {
		ids = append(ids, vpc.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

func (g *ServerGenerator) InitResources() error {
	client, err := g.generateClient()
	if err != nil {
		return err
	}
	return g.initResources(client)
}

func (g *ServerGenerator) initResources(client *govultr.Client) error {
	output, err := listAllVultrResources(context.Background(), client.Instance.List)
	if err != nil {
		return fmt.Errorf("list vultr instances: %w", err)
	}
	attachments, err := g.loadInstanceAttachments(client, output)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(output, attachments)
	return nil
}
