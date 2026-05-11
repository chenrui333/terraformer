// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	ipamResourceType                             = "aws_vpc_ipam"
	ipamScopeResourceType                        = "aws_vpc_ipam_scope"
	ipamPoolResourceType                         = "aws_vpc_ipam_pool"
	ipamPoolCIDRResourceType                     = "aws_vpc_ipam_pool_cidr"
	ipamResourceDiscoveryResourceType            = "aws_vpc_ipam_resource_discovery"
	ipamResourceDiscoveryAssociationResourceType = "aws_vpc_ipam_resource_discovery_association"
	ipamPoolCIDRImportIDSeparator                = "_"
)

var ipamAllowEmptyValues = []string{"tags."}

type IPAMGenerator struct {
	AWSService
}

func (g *IPAMGenerator) InitResources() error {
	config, err := g.generateConfig()
	if err != nil {
		return err
	}
	svc := ec2.NewFromConfig(config)

	loaders := []func(*ec2.Client) error{
		g.loadIPAMs,
		g.loadIPAMScopes,
		g.loadIPAMPools,
		g.loadIPAMPoolCIDRs,
		g.loadIPAMResourceDiscoveries,
		g.loadIPAMResourceDiscoveryAssociations,
	}
	for _, loader := range loaders {
		if err := loader(svc); err != nil {
			return err
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMs(svc *ec2.Client) error {
	p := ec2.NewDescribeIpamsPaginator(svc, &ec2.DescribeIpamsInput{
		Filters: g.ipamTagFilters("vpc_ipam"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ipam := range page.Ipams {
			if resource, ok := newIPAMResource(ipam); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMScopes(svc *ec2.Client) error {
	p := ec2.NewDescribeIpamScopesPaginator(svc, &ec2.DescribeIpamScopesInput{
		Filters: g.ipamTagFilters("vpc_ipam_scope"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, scope := range page.IpamScopes {
			if resource, ok := newIPAMScopeResource(scope); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMPools(svc *ec2.Client) error {
	p := ec2.NewDescribeIpamPoolsPaginator(svc, &ec2.DescribeIpamPoolsInput{
		Filters: g.ipamTagFilters("vpc_ipam_pool"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, pool := range page.IpamPools {
			if resource, ok := newIPAMPoolResource(pool); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMPoolCIDRs(svc *ec2.Client) error {
	pools := ec2.NewDescribeIpamPoolsPaginator(svc, &ec2.DescribeIpamPoolsInput{
		Filters: g.ipamTagFilters("vpc_ipam_pool"),
	})
	for pools.HasMorePages() {
		page, err := pools.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, pool := range page.IpamPools {
			if !ipamPoolImportable(pool) {
				continue
			}
			if err := g.loadIPAMPoolCIDRsForPool(svc, StringValue(pool.IpamPoolId)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMPoolCIDRsForPool(svc *ec2.Client, poolID string) error {
	if poolID == "" {
		return nil
	}
	p := ec2.NewGetIpamPoolCidrsPaginator(svc, &ec2.GetIpamPoolCidrsInput{
		IpamPoolId: aws.String(poolID),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, poolCIDR := range page.IpamPoolCidrs {
			if resource, ok := newIPAMPoolCIDRResource(poolID, poolCIDR); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMResourceDiscoveries(svc *ec2.Client) error {
	p := ec2.NewDescribeIpamResourceDiscoveriesPaginator(svc, &ec2.DescribeIpamResourceDiscoveriesInput{
		Filters: g.ipamTagFilters("vpc_ipam_resource_discovery"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, discovery := range page.IpamResourceDiscoveries {
			if resource, ok := newIPAMResourceDiscoveryResource(discovery); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) loadIPAMResourceDiscoveryAssociations(svc *ec2.Client) error {
	p := ec2.NewDescribeIpamResourceDiscoveryAssociationsPaginator(svc, &ec2.DescribeIpamResourceDiscoveryAssociationsInput{
		Filters: g.ipamTagFilters("vpc_ipam_resource_discovery_association"),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, association := range page.IpamResourceDiscoveryAssociations {
			if resource, ok := newIPAMResourceDiscoveryAssociationResource(association); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *IPAMGenerator) ipamTagFilters(resourceName string) []types.Filter {
	var filters []types.Filter
	for _, filter := range g.Filter {
		if strings.HasPrefix(filter.FieldPath, "tags.") && filter.IsApplicable(resourceName) {
			filters = append(filters, types.Filter{
				Name:   aws.String("tag:" + strings.TrimPrefix(filter.FieldPath, "tags.")),
				Values: filter.AcceptableValues,
			})
		}
	}
	return filters
}

func newIPAMResource(ipam types.Ipam) (terraformutils.Resource, bool) {
	if !ipamImportable(ipam) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(ipam.IpamId)
	return terraformutils.NewSimpleResource(
		id,
		ipamResourceName("ipam", StringValue(ipam.IpamRegion), id),
		ipamResourceType,
		"aws",
		ipamAllowEmptyValues,
	), true
}

func newIPAMScopeResource(scope types.IpamScope) (terraformutils.Resource, bool) {
	if !ipamScopeImportable(scope) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(scope.IpamScopeId)
	return terraformutils.NewSimpleResource(
		id,
		ipamResourceName("scope", StringValue(scope.IpamArn), string(scope.IpamScopeType), id),
		ipamScopeResourceType,
		"aws",
		ipamAllowEmptyValues,
	), true
}

func newIPAMPoolResource(pool types.IpamPool) (terraformutils.Resource, bool) {
	if !ipamPoolImportable(pool) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(pool.IpamPoolId)
	return terraformutils.NewSimpleResource(
		id,
		ipamResourceName("pool", StringValue(pool.IpamScopeArn), string(pool.AddressFamily), id),
		ipamPoolResourceType,
		"aws",
		ipamAllowEmptyValues,
	), true
}

func newIPAMPoolCIDRResource(poolID string, poolCIDR types.IpamPoolCidr) (terraformutils.Resource, bool) {
	if !ipamPoolCIDRImportable(poolID, poolCIDR) {
		return terraformutils.Resource{}, false
	}
	cidr := StringValue(poolCIDR.Cidr)
	importID := ipamPoolCIDRImportID(cidr, poolID)
	return terraformutils.NewResource(
		importID,
		ipamResourceName("pool_cidr", poolID, cidr),
		ipamPoolCIDRResourceType,
		"aws",
		map[string]string{
			"cidr":         cidr,
			"ipam_pool_id": poolID,
		},
		ipamAllowEmptyValues,
		map[string]interface{}{},
	), true
}

func newIPAMResourceDiscoveryResource(discovery types.IpamResourceDiscovery) (terraformutils.Resource, bool) {
	if !ipamResourceDiscoveryImportable(discovery) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(discovery.IpamResourceDiscoveryId)
	return terraformutils.NewSimpleResource(
		id,
		ipamResourceName("resource_discovery", StringValue(discovery.IpamResourceDiscoveryRegion), id),
		ipamResourceDiscoveryResourceType,
		"aws",
		ipamAllowEmptyValues,
	), true
}

func newIPAMResourceDiscoveryAssociationResource(association types.IpamResourceDiscoveryAssociation) (terraformutils.Resource, bool) {
	if !ipamResourceDiscoveryAssociationImportable(association) {
		return terraformutils.Resource{}, false
	}
	id := StringValue(association.IpamResourceDiscoveryAssociationId)
	return terraformutils.NewSimpleResource(
		id,
		ipamResourceName("resource_discovery_association", StringValue(association.IpamId), StringValue(association.IpamResourceDiscoveryId), id),
		ipamResourceDiscoveryAssociationResourceType,
		"aws",
		ipamAllowEmptyValues,
	), true
}

func ipamImportable(ipam types.Ipam) bool {
	if StringValue(ipam.IpamId) == "" {
		return false
	}
	switch ipam.State {
	case types.IpamStateCreateComplete, types.IpamStateModifyComplete:
		return true
	default:
		return false
	}
}

func ipamScopeImportable(scope types.IpamScope) bool {
	if StringValue(scope.IpamScopeId) == "" || aws.ToBool(scope.IsDefault) {
		return false
	}
	switch scope.State {
	case types.IpamScopeStateCreateComplete, types.IpamScopeStateModifyComplete:
		return true
	default:
		return false
	}
}

func ipamPoolImportable(pool types.IpamPool) bool {
	if StringValue(pool.IpamPoolId) == "" {
		return false
	}
	switch pool.State {
	case types.IpamPoolStateCreateComplete, types.IpamPoolStateModifyComplete:
		return true
	default:
		return false
	}
}

func ipamPoolCIDRImportable(poolID string, poolCIDR types.IpamPoolCidr) bool {
	if poolID == "" || StringValue(poolCIDR.Cidr) == "" {
		return false
	}
	return poolCIDR.State == types.IpamPoolCidrStateProvisioned
}

func ipamResourceDiscoveryImportable(discovery types.IpamResourceDiscovery) bool {
	if StringValue(discovery.IpamResourceDiscoveryId) == "" || aws.ToBool(discovery.IsDefault) {
		return false
	}
	switch discovery.State {
	case types.IpamResourceDiscoveryStateCreateComplete, types.IpamResourceDiscoveryStateModifyComplete:
		return true
	default:
		return false
	}
}

func ipamResourceDiscoveryAssociationImportable(association types.IpamResourceDiscoveryAssociation) bool {
	if StringValue(association.IpamResourceDiscoveryAssociationId) == "" || aws.ToBool(association.IsDefault) {
		return false
	}
	if association.ResourceDiscoveryStatus != types.IpamAssociatedResourceDiscoveryStatusActive {
		return false
	}
	return association.State == types.IpamResourceDiscoveryAssociationStateAssociateComplete
}

func ipamPoolCIDRImportID(cidr, poolID string) string {
	if cidr == "" || poolID == "" {
		return ""
	}
	return cidr + ipamPoolCIDRImportIDSeparator + poolID
}

func ipamResourceName(parts ...string) string {
	nameParts := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		if part == "" {
			continue
		}
		nameParts = append(nameParts, strconv.Itoa(len(part)), part)
	}
	return strings.Join(nameParts, "_")
}
