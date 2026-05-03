// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appmesh"
	appmeshtypes "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	appMeshMeshResourceType           = "appmesh_mesh"
	appMeshRouteResourceType          = "appmesh_route"
	appMeshVirtualGatewayResourceType = "appmesh_virtual_gateway"
	appMeshVirtualNodeResourceType    = "appmesh_virtual_node"
	appMeshVirtualRouterResourceType  = "appmesh_virtual_router"
	appMeshVirtualServiceResourceType = "appmesh_virtual_service"
	appMeshGatewayRouteResourceType   = "appmesh_gateway_route"

	appMeshIDSeparator = "/"
)

var (
	appMeshAllowEmptyValues = []string{"tags."}

	appMeshChildResourceTypes = []string{
		appMeshGatewayRouteResourceType,
		appMeshRouteResourceType,
		appMeshVirtualGatewayResourceType,
		appMeshVirtualNodeResourceType,
		appMeshVirtualRouterResourceType,
		appMeshVirtualServiceResourceType,
	}
	appMeshResourceTypes = append([]string{appMeshMeshResourceType}, appMeshChildResourceTypes...)
)

type AppMeshGenerator struct {
	AWSService
}

func (g *AppMeshGenerator) InitialCleanup() {
	if len(g.Filter) == 0 {
		return
	}
	filteredResources := []terraformutils.Resource{}
	for _, resource := range g.Resources {
		serviceName := strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")
		if g.hasTypedAppMeshFilter() && !g.hasTypedFilterFor(serviceName) && !g.hasUntypedIDFilter() {
			continue
		}
		allPredicatesTrue := true
		for _, filter := range g.Filter {
			if filter.FieldPath != "id" {
				continue
			}
			allPredicatesTrue = allPredicatesTrue && appMeshInitialIDFilterMatchesResource(filter, resource)
		}
		if allPredicatesTrue && !terraformutils.ContainsResource(filteredResources, resource) {
			filteredResources = append(filteredResources, resource)
		}
	}
	g.Resources = filteredResources
}

func (g *AppMeshGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := appmesh.NewFromConfig(config)

	if !g.shouldLoadMeshes() {
		return nil
	}
	return g.loadMeshes(svc)
}

func (g *AppMeshGenerator) loadMeshes(svc *appmesh.Client) error {
	p := appmesh.NewListMeshesPaginator(svc, &appmesh.ListMeshesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, meshRef := range page.Meshes {
			meshName := StringValue(meshRef.MeshName)
			meshOwner := StringValue(meshRef.MeshOwner)
			if meshName == "" {
				continue
			}
			mesh, err := g.describeMesh(svc, meshName, meshOwner)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if mesh == nil {
				continue
			}
			meshResource := newAppMeshMeshResource(meshName, meshOwner)
			if g.shouldAppendMeshResource(meshResource) {
				g.Resources = append(g.Resources, meshResource)
			}
			if !g.shouldLoadMeshChildren(meshResource) {
				continue
			}
			if g.shouldLoadVirtualNodes(meshName) {
				if err := g.loadVirtualNodes(svc, meshName, meshOwner); err != nil {
					return err
				}
			}
			if g.shouldLoadVirtualRouters(meshName) {
				if err := g.loadVirtualRouters(svc, meshName, meshOwner); err != nil {
					return err
				}
			}
			if g.shouldLoadVirtualServices(meshName) {
				if err := g.loadVirtualServices(svc, meshName, meshOwner); err != nil {
					return err
				}
			}
			if g.shouldLoadVirtualGateways(meshName) {
				if err := g.loadVirtualGateways(svc, meshName, meshOwner); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadVirtualNodes(svc *appmesh.Client, meshName, meshOwner string) error {
	p := appmesh.NewListVirtualNodesPaginator(svc, &appmesh.ListVirtualNodesInput{
		MeshName:  appMeshString(meshName),
		MeshOwner: appMeshOptionalString(meshOwner),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.VirtualNodes {
			name := StringValue(ref.VirtualNodeName)
			if name == "" {
				continue
			}
			virtualNode, err := g.describeVirtualNode(svc, meshName, meshOwner, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if virtualNode == nil || !appMeshVirtualNodeImportable(virtualNode) {
				continue
			}
			resource := newAppMeshVirtualNodeResource(meshName, meshOwner, name, appMeshResourceUID(virtualNode.Metadata))
			if g.shouldAppendMeshChildResource(appMeshVirtualNodeResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadVirtualRouters(svc *appmesh.Client, meshName, meshOwner string) error {
	p := appmesh.NewListVirtualRoutersPaginator(svc, &appmesh.ListVirtualRoutersInput{
		MeshName:  appMeshString(meshName),
		MeshOwner: appMeshOptionalString(meshOwner),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.VirtualRouters {
			name := StringValue(ref.VirtualRouterName)
			if name == "" {
				continue
			}
			virtualRouter, err := g.describeVirtualRouter(svc, meshName, meshOwner, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if virtualRouter == nil || !appMeshVirtualRouterImportable(virtualRouter) {
				continue
			}
			resource := newAppMeshVirtualRouterResource(meshName, meshOwner, name, appMeshResourceUID(virtualRouter.Metadata))
			if g.shouldAppendMeshChildResource(appMeshVirtualRouterResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
			if g.shouldLoadRoutes(meshName, name) {
				if err := g.loadRoutes(svc, meshName, meshOwner, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadRoutes(svc *appmesh.Client, meshName, meshOwner, virtualRouterName string) error {
	p := appmesh.NewListRoutesPaginator(svc, &appmesh.ListRoutesInput{
		MeshName:          appMeshString(meshName),
		MeshOwner:         appMeshOptionalString(meshOwner),
		VirtualRouterName: appMeshString(virtualRouterName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.Routes {
			name := StringValue(ref.RouteName)
			if name == "" {
				continue
			}
			route, err := g.describeRoute(svc, meshName, meshOwner, virtualRouterName, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if route == nil || !appMeshRouteImportable(route) {
				continue
			}
			resource := newAppMeshRouteResource(meshName, meshOwner, virtualRouterName, name, appMeshResourceUID(route.Metadata))
			if g.shouldAppendMeshChildResource(appMeshRouteResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadVirtualServices(svc *appmesh.Client, meshName, meshOwner string) error {
	p := appmesh.NewListVirtualServicesPaginator(svc, &appmesh.ListVirtualServicesInput{
		MeshName:  appMeshString(meshName),
		MeshOwner: appMeshOptionalString(meshOwner),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.VirtualServices {
			name := StringValue(ref.VirtualServiceName)
			if name == "" {
				continue
			}
			virtualService, err := g.describeVirtualService(svc, meshName, meshOwner, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if virtualService == nil || !appMeshVirtualServiceImportable(virtualService) {
				continue
			}
			resource := newAppMeshVirtualServiceResource(meshName, meshOwner, name, appMeshResourceUID(virtualService.Metadata))
			if g.shouldAppendMeshChildResource(appMeshVirtualServiceResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadVirtualGateways(svc *appmesh.Client, meshName, meshOwner string) error {
	p := appmesh.NewListVirtualGatewaysPaginator(svc, &appmesh.ListVirtualGatewaysInput{
		MeshName:  appMeshString(meshName),
		MeshOwner: appMeshOptionalString(meshOwner),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.VirtualGateways {
			name := StringValue(ref.VirtualGatewayName)
			if name == "" {
				continue
			}
			virtualGateway, err := g.describeVirtualGateway(svc, meshName, meshOwner, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if virtualGateway == nil || !appMeshVirtualGatewayImportable(virtualGateway) {
				continue
			}
			resource := newAppMeshVirtualGatewayResource(meshName, meshOwner, name, appMeshResourceUID(virtualGateway.Metadata))
			if g.shouldAppendMeshChildResource(appMeshVirtualGatewayResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
			if g.shouldLoadGatewayRoutes(meshName, name) {
				if err := g.loadGatewayRoutes(svc, meshName, meshOwner, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) loadGatewayRoutes(svc *appmesh.Client, meshName, meshOwner, virtualGatewayName string) error {
	p := appmesh.NewListGatewayRoutesPaginator(svc, &appmesh.ListGatewayRoutesInput{
		MeshName:           appMeshString(meshName),
		MeshOwner:          appMeshOptionalString(meshOwner),
		VirtualGatewayName: appMeshString(virtualGatewayName),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			if appMeshResourceMissing(err) {
				return nil
			}
			return err
		}
		for _, ref := range page.GatewayRoutes {
			name := StringValue(ref.GatewayRouteName)
			if name == "" {
				continue
			}
			gatewayRoute, err := g.describeGatewayRoute(svc, meshName, meshOwner, virtualGatewayName, name)
			if err != nil {
				if appMeshResourceMissing(err) {
					continue
				}
				return err
			}
			if gatewayRoute == nil || !appMeshGatewayRouteImportable(gatewayRoute) {
				continue
			}
			resource := newAppMeshGatewayRouteResource(meshName, meshOwner, virtualGatewayName, name, appMeshResourceUID(gatewayRoute.Metadata))
			if g.shouldAppendMeshChildResource(appMeshGatewayRouteResourceType, resource) {
				g.Resources = append(g.Resources, resource)
			}
		}
	}
	return nil
}

func (g *AppMeshGenerator) describeMesh(svc *appmesh.Client, meshName, meshOwner string) (*appmeshtypes.MeshData, error) {
	output, err := svc.DescribeMesh(context.TODO(), &appmesh.DescribeMeshInput{
		MeshName:  appMeshString(meshName),
		MeshOwner: appMeshOptionalString(meshOwner),
	})
	if err != nil || output == nil || output.Mesh == nil {
		return nil, err
	}
	if !appMeshMeshImportable(output.Mesh) {
		return nil, nil
	}
	return output.Mesh, nil
}

func (g *AppMeshGenerator) describeVirtualNode(svc *appmesh.Client, meshName, meshOwner, name string) (*appmeshtypes.VirtualNodeData, error) {
	output, err := svc.DescribeVirtualNode(context.TODO(), &appmesh.DescribeVirtualNodeInput{
		MeshName:        appMeshString(meshName),
		MeshOwner:       appMeshOptionalString(meshOwner),
		VirtualNodeName: appMeshString(name),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.VirtualNode, nil
}

func (g *AppMeshGenerator) describeVirtualRouter(svc *appmesh.Client, meshName, meshOwner, name string) (*appmeshtypes.VirtualRouterData, error) {
	output, err := svc.DescribeVirtualRouter(context.TODO(), &appmesh.DescribeVirtualRouterInput{
		MeshName:          appMeshString(meshName),
		MeshOwner:         appMeshOptionalString(meshOwner),
		VirtualRouterName: appMeshString(name),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.VirtualRouter, nil
}

func (g *AppMeshGenerator) describeRoute(svc *appmesh.Client, meshName, meshOwner, virtualRouterName, name string) (*appmeshtypes.RouteData, error) {
	output, err := svc.DescribeRoute(context.TODO(), &appmesh.DescribeRouteInput{
		MeshName:          appMeshString(meshName),
		MeshOwner:         appMeshOptionalString(meshOwner),
		RouteName:         appMeshString(name),
		VirtualRouterName: appMeshString(virtualRouterName),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.Route, nil
}

func (g *AppMeshGenerator) describeVirtualService(svc *appmesh.Client, meshName, meshOwner, name string) (*appmeshtypes.VirtualServiceData, error) {
	output, err := svc.DescribeVirtualService(context.TODO(), &appmesh.DescribeVirtualServiceInput{
		MeshName:           appMeshString(meshName),
		MeshOwner:          appMeshOptionalString(meshOwner),
		VirtualServiceName: appMeshString(name),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.VirtualService, nil
}

func (g *AppMeshGenerator) describeVirtualGateway(svc *appmesh.Client, meshName, meshOwner, name string) (*appmeshtypes.VirtualGatewayData, error) {
	output, err := svc.DescribeVirtualGateway(context.TODO(), &appmesh.DescribeVirtualGatewayInput{
		MeshName:           appMeshString(meshName),
		MeshOwner:          appMeshOptionalString(meshOwner),
		VirtualGatewayName: appMeshString(name),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.VirtualGateway, nil
}

func (g *AppMeshGenerator) describeGatewayRoute(svc *appmesh.Client, meshName, meshOwner, virtualGatewayName, name string) (*appmeshtypes.GatewayRouteData, error) {
	output, err := svc.DescribeGatewayRoute(context.TODO(), &appmesh.DescribeGatewayRouteInput{
		GatewayRouteName:   appMeshString(name),
		MeshName:           appMeshString(meshName),
		MeshOwner:          appMeshOptionalString(meshOwner),
		VirtualGatewayName: appMeshString(virtualGatewayName),
	})
	if err != nil || output == nil {
		return nil, err
	}
	return output.GatewayRoute, nil
}

func newAppMeshMeshResource(meshName, meshOwner string) terraformutils.Resource {
	attributes := map[string]string{"name": meshName}
	if meshOwner != "" {
		attributes["mesh_owner"] = meshOwner
	}
	return terraformutils.NewResource(
		meshName,
		appMeshResourceNameWithOwner(meshOwner, meshName),
		"aws_appmesh_mesh",
		"aws",
		attributes,
		appMeshAllowEmptyValues,
		map[string]interface{}{},
	)
}

func newAppMeshVirtualNodeResource(meshName, meshOwner, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, name), "aws_appmesh_virtual_node", meshName, meshOwner, name, "")
}

func newAppMeshVirtualRouterResource(meshName, meshOwner, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, name), "aws_appmesh_virtual_router", meshName, meshOwner, name, "")
}

func newAppMeshRouteResource(meshName, meshOwner, virtualRouterName, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, virtualRouterName, name), "aws_appmesh_route", meshName, meshOwner, name, virtualRouterName)
}

func newAppMeshVirtualServiceResource(meshName, meshOwner, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, name), "aws_appmesh_virtual_service", meshName, meshOwner, name, "")
}

func newAppMeshVirtualGatewayResource(meshName, meshOwner, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, name), "aws_appmesh_virtual_gateway", meshName, meshOwner, name, "")
}

func newAppMeshGatewayRouteResource(meshName, meshOwner, virtualGatewayName, name, uid string) terraformutils.Resource {
	return newAppMeshNamedChildResource(uid, appMeshResourceNameWithOwner(meshOwner, meshName, virtualGatewayName, name), "aws_appmesh_gateway_route", meshName, meshOwner, name, virtualGatewayName)
}

func newAppMeshNamedChildResource(id, resourceName, resourceType, meshName, meshOwner, name, parentName string) terraformutils.Resource {
	attributes := map[string]string{
		"mesh_name": meshName,
		"name":      name,
	}
	if meshOwner != "" {
		attributes["mesh_owner"] = meshOwner
	}
	switch resourceType {
	case "aws_appmesh_route":
		attributes["virtual_router_name"] = parentName
	case "aws_appmesh_gateway_route":
		attributes["virtual_gateway_name"] = parentName
	}
	return terraformutils.NewResource(
		id,
		resourceName,
		resourceType,
		"aws",
		attributes,
		appMeshAllowEmptyValues,
		map[string]interface{}{},
	)
}

func appMeshResourceName(parts ...string) string {
	nonEmptyParts := []string{}
	for _, part := range parts {
		if part != "" {
			nonEmptyParts = append(nonEmptyParts, part)
		}
	}
	return strings.Join(nonEmptyParts, ":")
}

func appMeshResourceNameWithOwner(meshOwner string, parts ...string) string {
	if meshOwner == "" {
		return appMeshResourceName(parts...)
	}
	return appMeshResourceName(append([]string{meshOwner}, parts...)...)
}

func appMeshResourceUID(metadata *appmeshtypes.ResourceMetadata) string {
	if metadata == nil {
		return ""
	}
	return StringValue(metadata.Uid)
}

func appMeshString(value string) *string {
	return aws.String(value)
}

func appMeshOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return aws.String(value)
}

func appMeshResourceMissing(err error) bool {
	var notFound *appmeshtypes.NotFoundException
	return errors.As(err, &notFound)
}

func appMeshMeshImportable(mesh *appmeshtypes.MeshData) bool {
	return mesh != nil && (mesh.Status == nil || mesh.Status.Status != appmeshtypes.MeshStatusCodeDeleted)
}

func appMeshVirtualNodeImportable(virtualNode *appmeshtypes.VirtualNodeData) bool {
	return virtualNode != nil && (virtualNode.Status == nil || virtualNode.Status.Status != appmeshtypes.VirtualNodeStatusCodeDeleted)
}

func appMeshVirtualRouterImportable(virtualRouter *appmeshtypes.VirtualRouterData) bool {
	return virtualRouter != nil && (virtualRouter.Status == nil || virtualRouter.Status.Status != appmeshtypes.VirtualRouterStatusCodeDeleted)
}

func appMeshRouteImportable(route *appmeshtypes.RouteData) bool {
	return route != nil && (route.Status == nil || route.Status.Status != appmeshtypes.RouteStatusCodeDeleted)
}

func appMeshVirtualServiceImportable(virtualService *appmeshtypes.VirtualServiceData) bool {
	return virtualService != nil && (virtualService.Status == nil || virtualService.Status.Status != appmeshtypes.VirtualServiceStatusCodeDeleted)
}

func appMeshVirtualGatewayImportable(virtualGateway *appmeshtypes.VirtualGatewayData) bool {
	return virtualGateway != nil && (virtualGateway.Status == nil || virtualGateway.Status.Status != appmeshtypes.VirtualGatewayStatusCodeDeleted)
}

func appMeshGatewayRouteImportable(gatewayRoute *appmeshtypes.GatewayRouteData) bool {
	return gatewayRoute != nil && (gatewayRoute.Status == nil || gatewayRoute.Status.Status != appmeshtypes.GatewayRouteStatusCodeDeleted)
}

func (g *AppMeshGenerator) shouldLoadMeshes() bool {
	if g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppMeshFilter() {
		return g.hasTypedFilterFor(appMeshMeshResourceType) || g.hasTypedAppMeshChildFilter()
	}
	return true
}

func (g *AppMeshGenerator) shouldLoadMeshChildren(meshResource terraformutils.Resource) bool {
	if !g.hasTypedAppMeshFilter() && !g.hasUntypedIDFilter() {
		return true
	}
	if g.hasTypedAppMeshFilter() && !g.hasTypedAppMeshChildFilter() && !g.hasUntypedIDFilter() {
		return false
	}
	meshName := meshResource.InstanceState.ID
	for _, childServiceName := range appMeshChildResourceTypes {
		if g.shouldLoadMeshChildResourceType(childServiceName, meshName) {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) shouldLoadVirtualNodes(meshName string) bool {
	return g.shouldLoadMeshChildResourceType(appMeshVirtualNodeResourceType, meshName)
}

func (g *AppMeshGenerator) shouldLoadVirtualRouters(meshName string) bool {
	return g.shouldLoadMeshChildResourceType(appMeshVirtualRouterResourceType, meshName) ||
		g.shouldLoadMeshChildResourceType(appMeshRouteResourceType, meshName)
}

func (g *AppMeshGenerator) shouldLoadRoutes(meshName, virtualRouterName string) bool {
	if !g.shouldLoadMeshChildResourceType(appMeshRouteResourceType, meshName) {
		return false
	}
	return g.initialIDFiltersCanMatchNestedChild(appMeshRouteResourceType, meshName, virtualRouterName)
}

func (g *AppMeshGenerator) shouldLoadVirtualServices(meshName string) bool {
	return g.shouldLoadMeshChildResourceType(appMeshVirtualServiceResourceType, meshName)
}

func (g *AppMeshGenerator) shouldLoadVirtualGateways(meshName string) bool {
	return g.shouldLoadMeshChildResourceType(appMeshVirtualGatewayResourceType, meshName) ||
		g.shouldLoadMeshChildResourceType(appMeshGatewayRouteResourceType, meshName)
}

func (g *AppMeshGenerator) shouldLoadGatewayRoutes(meshName, virtualGatewayName string) bool {
	if !g.shouldLoadMeshChildResourceType(appMeshGatewayRouteResourceType, meshName) {
		return false
	}
	return g.initialIDFiltersCanMatchNestedChild(appMeshGatewayRouteResourceType, meshName, virtualGatewayName)
}

func (g *AppMeshGenerator) shouldLoadMeshChildResourceType(serviceName, meshName string) bool {
	hasTypedChildFilter := g.hasTypedFilterFor(serviceName)
	if g.hasTypedAppMeshChildFilter() && !hasTypedChildFilter {
		return false
	}
	if g.hasTypedAppMeshFilter() && !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return false
	}
	if !g.initialIDFiltersCanMatchMeshChild(serviceName, meshName) {
		return false
	}
	if !hasTypedChildFilter && !g.hasUntypedIDFilter() {
		return g.meshMatchesPreDiscoveryFilters(meshName)
	}
	if hasTypedChildFilter && !g.hasIDFilterFor(serviceName) && !g.hasUntypedIDFilter() && g.hasTypedIDFilterFor(appMeshMeshResourceType) {
		return g.meshMatchesInitialIDFilters(meshName)
	}
	return true
}

func (g *AppMeshGenerator) meshMatchesPreDiscoveryFilters(meshName string) bool {
	if !g.meshMatchesInitialIDFilters(meshName) {
		return false
	}
	return !g.hasTypedNonIDFilterFor(appMeshMeshResourceType)
}

func (g *AppMeshGenerator) meshMatchesInitialIDFilters(meshName string) bool {
	meshResource := newAppMeshMeshResource(meshName, "")
	return g.resourceMatchesInitialIDFilters(appMeshMeshResourceType, meshResource)
}

func (g *AppMeshGenerator) shouldAppendMeshResource(resource terraformutils.Resource) bool {
	if !g.resourceMatchesInitialIDFilters(appMeshMeshResourceType, resource) {
		return false
	}
	if g.hasTypedAppMeshFilter() && !g.hasTypedFilterFor(appMeshMeshResourceType) && !g.hasUntypedIDFilter() {
		return false
	}
	return true
}

func (g *AppMeshGenerator) shouldAppendMeshChildResource(serviceName string, resource terraformutils.Resource) bool {
	if g.hasTypedAppMeshChildFilter() && !g.hasTypedFilterFor(serviceName) {
		return false
	}
	if g.hasTypedAppMeshFilter() && !g.hasTypedAppMeshChildFilter() && !g.hasUntypedIDFilter() {
		return false
	}
	return g.resourceMatchesInitialIDFilters(serviceName, resource)
}

func (g *AppMeshGenerator) resourceMatchesInitialIDFilters(serviceName string, resource terraformutils.Resource) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appMeshInitialIDFilterMatchesResource(filter, resource) {
			return false
		}
	}
	return true
}

func (g *AppMeshGenerator) initialIDFiltersCanMatchMeshChild(serviceName, meshName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appMeshAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			return appMeshChildIDMayBelongToMesh(serviceName, meshName, value)
		}) {
			return false
		}
	}
	return true
}

func (g *AppMeshGenerator) initialIDFiltersCanMatchNestedChild(serviceName, meshName, parentName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable(serviceName) {
			continue
		}
		if !appMeshAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
			parts := strings.Split(value, appMeshIDSeparator)
			switch serviceName {
			case appMeshRouteResourceType:
				if len(parts) != 3 {
					return true
				}
				return parts[0] == meshName && parts[1] == parentName
			case appMeshGatewayRouteResourceType:
				if len(parts) != 3 {
					return true
				}
				return parts[0] == meshName && parts[1] == parentName
			default:
				return true
			}
		}) {
			return false
		}
	}
	return true
}

func appMeshInitialIDFilterMatchesResource(filter terraformutils.ResourceFilter, resource terraformutils.Resource) bool {
	serviceName := strings.TrimPrefix(resource.InstanceInfo.Type, resource.Provider+"_")
	if !filter.IsApplicable(serviceName) {
		return true
	}
	if filter.Filter(resource) {
		return true
	}
	importID, ok := appMeshResourceImportID(serviceName, resource)
	if !ok {
		return false
	}
	return appMeshAnyAcceptableIDMatches(filter.AcceptableValues, func(value string) bool {
		return value == importID
	})
}

func appMeshResourceImportID(serviceName string, resource terraformutils.Resource) (string, bool) {
	attributes := resource.InstanceState.Attributes
	meshName := attributes["mesh_name"]
	name := attributes["name"]
	switch serviceName {
	case appMeshMeshResourceType:
		if resource.InstanceState.ID == "" {
			return "", false
		}
		return resource.InstanceState.ID, true
	case appMeshVirtualNodeResourceType, appMeshVirtualRouterResourceType, appMeshVirtualServiceResourceType, appMeshVirtualGatewayResourceType:
		if meshName == "" || name == "" {
			return "", false
		}
		return strings.Join([]string{meshName, name}, appMeshIDSeparator), true
	case appMeshRouteResourceType:
		virtualRouterName := attributes["virtual_router_name"]
		if meshName == "" || virtualRouterName == "" || name == "" {
			return "", false
		}
		return strings.Join([]string{meshName, virtualRouterName, name}, appMeshIDSeparator), true
	case appMeshGatewayRouteResourceType:
		virtualGatewayName := attributes["virtual_gateway_name"]
		if meshName == "" || virtualGatewayName == "" || name == "" {
			return "", false
		}
		return strings.Join([]string{meshName, virtualGatewayName, name}, appMeshIDSeparator), true
	default:
		return "", false
	}
}

func appMeshChildIDMayBelongToMesh(serviceName, meshName, value string) bool {
	parts := strings.Split(value, appMeshIDSeparator)
	switch serviceName {
	case appMeshVirtualNodeResourceType, appMeshVirtualRouterResourceType, appMeshVirtualServiceResourceType, appMeshVirtualGatewayResourceType:
		if len(parts) != 2 {
			return true
		}
		return parts[0] == meshName
	case appMeshRouteResourceType, appMeshGatewayRouteResourceType:
		if len(parts) != 3 {
			return true
		}
		return parts[0] == meshName
	default:
		return value == meshName
	}
}

func appMeshAnyAcceptableIDMatches(values []string, match func(string) bool) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasTypedAppMeshChildFilter() bool {
	for _, serviceName := range appMeshChildResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasTypedAppMeshFilter() bool {
	for _, serviceName := range appMeshResourceTypes {
		if g.hasTypedFilterFor(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasTypedFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasTypedNonIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName && filter.FieldPath != "id" {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" && filter.IsApplicable(serviceName) {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasTypedIDFilterFor(serviceName string) bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == serviceName && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}

func (g *AppMeshGenerator) hasUntypedIDFilter() bool {
	for _, filter := range g.Filter {
		if filter.ServiceName == "" && filter.FieldPath == "id" {
			return true
		}
	}
	return false
}
