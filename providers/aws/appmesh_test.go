// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	appmeshtypes "github.com/aws/aws-sdk-go-v2/service/appmesh/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestAppMeshResourceImportID(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		resource    terraformutils.Resource
		want        string
	}{
		{
			name:        "mesh",
			serviceName: appMeshMeshResourceType,
			resource:    newAppMeshMeshResource("orders", "123456789012"),
			want:        "orders",
		},
		{
			name:        "virtual node",
			serviceName: appMeshVirtualNodeResourceType,
			resource:    newAppMeshVirtualNodeResource("orders", "123456789012", "api", "uid-node"),
			want:        "orders/api",
		},
		{
			name:        "route",
			serviceName: appMeshRouteResourceType,
			resource:    newAppMeshRouteResource("orders", "123456789012", "router", "route", "uid-route"),
			want:        "orders/router/route",
		},
		{
			name:        "gateway route",
			serviceName: appMeshGatewayRouteResourceType,
			resource:    newAppMeshGatewayRouteResource("orders", "123456789012", "gateway", "route", "uid-gateway-route"),
			want:        "orders/gateway/route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := appMeshResourceImportID(tt.serviceName, tt.resource)
			if !ok {
				t.Fatal("appMeshResourceImportID returned !ok")
			}
			if got != tt.want {
				t.Fatalf("appMeshResourceImportID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAppMeshChildrenSeedProviderUIDStateID(t *testing.T) {
	node := newAppMeshVirtualNodeResource("orders", "", "api", "uid-node")
	if got := node.InstanceState.ID; got != "uid-node" {
		t.Fatalf("virtual node state ID = %q, want uid-node", got)
	}
	if got := node.InstanceState.Attributes["mesh_name"]; got != "orders" {
		t.Fatalf("virtual node mesh_name = %q, want orders", got)
	}
	if got := node.InstanceState.Attributes["name"]; got != "api" {
		t.Fatalf("virtual node name = %q, want api", got)
	}
}

func TestAppMeshResourceNamesIncludeParentNames(t *testing.T) {
	node := newAppMeshVirtualNodeResource("orders", "", "api", "uid-node")
	otherNode := newAppMeshVirtualNodeResource("payments", "", "api", "uid-other-node")
	if node.ResourceName == otherNode.ResourceName {
		t.Fatalf("virtual node resource names collide: %q", node.ResourceName)
	}

	route := newAppMeshRouteResource("orders", "", "router", "default", "uid-route")
	otherRoute := newAppMeshRouteResource("orders", "", "other-router", "default", "uid-other-route")
	if route.ResourceName == otherRoute.ResourceName {
		t.Fatalf("route resource names collide: %q", route.ResourceName)
	}

	ownedMesh := newAppMeshMeshResource("orders", "111111111111")
	sharedMesh := newAppMeshMeshResource("orders", "222222222222")
	if ownedMesh.ResourceName == sharedMesh.ResourceName {
		t.Fatalf("shared mesh resource names collide: %q", ownedMesh.ResourceName)
	}

	ownedNode := newAppMeshVirtualNodeResource("orders", "111111111111", "api", "uid-owned-node")
	sharedNode := newAppMeshVirtualNodeResource("orders", "222222222222", "api", "uid-shared-node")
	if ownedNode.ResourceName == sharedNode.ResourceName {
		t.Fatalf("shared virtual node resource names collide: %q", ownedNode.ResourceName)
	}
}

func TestAppMeshResourceMissing(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "not found", err: &appmeshtypes.NotFoundException{}, want: true},
		{name: "wrapped not found", err: errors.Join(errors.New("boom"), &appmeshtypes.NotFoundException{}), want: true},
		{name: "generic", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appMeshResourceMissing(tt.err); got != tt.want {
				t.Fatalf("appMeshResourceMissing(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func TestAppMeshSkipsDeletedResources(t *testing.T) {
	if appMeshMeshImportable(&appmeshtypes.MeshData{Status: &appmeshtypes.MeshStatus{Status: appmeshtypes.MeshStatusCodeDeleted}}) {
		t.Fatal("deleted mesh should not be importable")
	}
	if !appMeshMeshImportable(&appmeshtypes.MeshData{Status: &appmeshtypes.MeshStatus{Status: appmeshtypes.MeshStatusCodeInactive}}) {
		t.Fatal("inactive mesh should remain importable")
	}
	if appMeshRouteImportable(&appmeshtypes.RouteData{Status: &appmeshtypes.RouteStatus{Status: appmeshtypes.RouteStatusCodeDeleted}}) {
		t.Fatal("deleted route should not be importable")
	}
	if !appMeshRouteImportable(&appmeshtypes.RouteData{Status: &appmeshtypes.RouteStatus{Status: appmeshtypes.RouteStatusCodeActive}}) {
		t.Fatal("active route should be importable")
	}
}

func TestAppMeshInitialCleanupPreservesImportIDs(t *testing.T) {
	mesh := newAppMeshMeshResource("orders", "")
	node := newAppMeshVirtualNodeResource("orders", "", "api", "uid-node")
	route := newAppMeshRouteResource("orders", "", "router", "default", "uid-route")

	tests := []struct {
		name      string
		filters   []terraformutils.ResourceFilter
		resources []terraformutils.Resource
		wantIDs   []string
	}{
		{
			name: "typed child import id keeps UID-backed child",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshVirtualNodeResourceType, FieldPath: "id", AcceptableValues: []string{"orders/api"}},
			},
			resources: []terraformutils.Resource{mesh, node, route},
			wantIDs:   []string{"uid-node"},
		},
		{
			name: "typed child UID also works",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshVirtualNodeResourceType, FieldPath: "id", AcceptableValues: []string{"uid-node"}},
			},
			resources: []terraformutils.Resource{mesh, node, route},
			wantIDs:   []string{"uid-node"},
		},
		{
			name: "untyped route import id keeps matching route",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{"orders/router/default"}},
			},
			resources: []terraformutils.Resource{mesh, node, route},
			wantIDs:   []string{"uid-route"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppMeshGenerator{}
			g.Filter = tt.filters
			g.Resources = append([]terraformutils.Resource{}, tt.resources...)
			g.InitialCleanup()
			if len(g.Resources) != len(tt.wantIDs) {
				t.Fatalf("resources len = %d, want %d", len(g.Resources), len(tt.wantIDs))
			}
			for i, wantID := range tt.wantIDs {
				if got := g.Resources[i].InstanceState.ID; got != wantID {
					t.Fatalf("resource[%d] ID = %q, want %q", i, got, wantID)
				}
			}
		})
	}
}

func TestAppMeshFilterGatesMeshAndChildDiscovery(t *testing.T) {
	meshName := "orders"
	otherMeshName := "payments"
	mesh := newAppMeshMeshResource(meshName, "123456789012")
	otherMesh := newAppMeshMeshResource(otherMeshName, "123456789012")
	node := newAppMeshVirtualNodeResource(meshName, "", "api", "uid-node")
	router := newAppMeshVirtualRouterResource(meshName, "", "router", "uid-router")
	route := newAppMeshRouteResource(meshName, "", "router", "default", "uid-route")
	gateway := newAppMeshVirtualGatewayResource(meshName, "", "gateway", "uid-gateway")
	gatewayRoute := newAppMeshGatewayRouteResource(meshName, "", "gateway", "default", "uid-gateway-route")

	tests := []struct {
		name                  string
		filters               []terraformutils.ResourceFilter
		loadMeshes            bool
		appendMesh            bool
		appendOtherMesh       bool
		loadChildren          bool
		loadOtherChildren     bool
		loadNodes             bool
		loadOtherNodes        bool
		loadRouters           bool
		loadRoutes            bool
		loadOtherRoutes       bool
		loadGateways          bool
		loadGatewayRoutes     bool
		loadOtherGatewayRoute bool
		appendNode            bool
		appendRouter          bool
		appendRoute           bool
		appendGateway         bool
		appendGatewayRoute    bool
	}{
		{
			name:                  "no filters imports meshes and children",
			loadMeshes:            true,
			appendMesh:            true,
			appendOtherMesh:       true,
			loadChildren:          true,
			loadOtherChildren:     true,
			loadNodes:             true,
			loadOtherNodes:        true,
			loadRouters:           true,
			loadRoutes:            true,
			loadOtherRoutes:       true,
			loadGateways:          true,
			loadGatewayRoutes:     true,
			loadOtherGatewayRoute: true,
			appendNode:            true,
			appendRouter:          true,
			appendRoute:           true,
			appendGateway:         true,
			appendGatewayRoute:    true,
		},
		{
			name: "typed mesh id filter imports only matching mesh",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshMeshResourceType, FieldPath: "id", AcceptableValues: []string{meshName}},
			},
			loadMeshes:   true,
			appendMesh:   true,
			loadChildren: false,
		},
		{
			name: "typed child id filter does not import parent mesh",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshVirtualNodeResourceType, FieldPath: "id", AcceptableValues: []string{"orders/api"}},
			},
			loadMeshes:   true,
			loadChildren: true,
			loadNodes:    true,
			appendNode:   true,
		},
		{
			name: "typed route id filter loads routers but only appends route",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshRouteResourceType, FieldPath: "id", AcceptableValues: []string{"orders/router/default"}},
			},
			loadMeshes:   true,
			loadChildren: true,
			loadRouters:  true,
			loadRoutes:   true,
			appendRoute:  true,
		},
		{
			name: "typed gateway route id filter loads gateways but only appends gateway route",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshGatewayRouteResourceType, FieldPath: "id", AcceptableValues: []string{"orders/gateway/default"}},
			},
			loadMeshes:         true,
			loadChildren:       true,
			loadGateways:       true,
			loadGatewayRoutes:  true,
			appendGatewayRoute: true,
		},
		{
			name: "typed mesh id scopes typed child non-id discovery",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshMeshResourceType, FieldPath: "id", AcceptableValues: []string{meshName}},
				{ServiceName: appMeshVirtualNodeResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadMeshes:   true,
			appendMesh:   true,
			loadChildren: true,
			loadNodes:    true,
			appendNode:   true,
		},
		{
			name: "typed mesh and child id filters load matching child outside parent filter",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshMeshResourceType, FieldPath: "id", AcceptableValues: []string{otherMeshName}},
				{ServiceName: appMeshVirtualNodeResourceType, FieldPath: "id", AcceptableValues: []string{"orders/api"}},
			},
			loadMeshes:      true,
			appendOtherMesh: true,
			loadChildren:    true,
			loadNodes:       true,
			appendNode:      true,
		},
		{
			name: "typed mesh non-id filter avoids child pre-load",
			filters: []terraformutils.ResourceFilter{
				{ServiceName: appMeshMeshResourceType, FieldPath: "tags.env", AcceptableValues: []string{"prod"}},
			},
			loadMeshes:      true,
			appendMesh:      true,
			appendOtherMesh: true,
		},
		{
			name: "untyped mesh id filter avoids child scans",
			filters: []terraformutils.ResourceFilter{
				{FieldPath: "id", AcceptableValues: []string{meshName}},
			},
			loadMeshes:   true,
			appendMesh:   true,
			loadChildren: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := AppMeshGenerator{}
			g.Filter = tt.filters
			if got := g.shouldLoadMeshes(); got != tt.loadMeshes {
				t.Fatalf("shouldLoadMeshes() = %t, want %t", got, tt.loadMeshes)
			}
			if got := g.shouldAppendMeshResource(mesh); got != tt.appendMesh {
				t.Fatalf("shouldAppendMeshResource(mesh) = %t, want %t", got, tt.appendMesh)
			}
			if got := g.shouldAppendMeshResource(otherMesh); got != tt.appendOtherMesh {
				t.Fatalf("shouldAppendMeshResource(other mesh) = %t, want %t", got, tt.appendOtherMesh)
			}
			if got := g.shouldLoadMeshChildren(mesh); got != tt.loadChildren {
				t.Fatalf("shouldLoadMeshChildren(mesh) = %t, want %t", got, tt.loadChildren)
			}
			if got := g.shouldLoadMeshChildren(otherMesh); got != tt.loadOtherChildren {
				t.Fatalf("shouldLoadMeshChildren(other mesh) = %t, want %t", got, tt.loadOtherChildren)
			}
			if got := g.shouldLoadVirtualNodes(meshName); got != tt.loadNodes {
				t.Fatalf("shouldLoadVirtualNodes(mesh) = %t, want %t", got, tt.loadNodes)
			}
			if got := g.shouldLoadVirtualNodes(otherMeshName); got != tt.loadOtherNodes {
				t.Fatalf("shouldLoadVirtualNodes(other mesh) = %t, want %t", got, tt.loadOtherNodes)
			}
			if got := g.shouldLoadVirtualRouters(meshName); got != tt.loadRouters {
				t.Fatalf("shouldLoadVirtualRouters(mesh) = %t, want %t", got, tt.loadRouters)
			}
			if got := g.shouldLoadRoutes(meshName, "router"); got != tt.loadRoutes {
				t.Fatalf("shouldLoadRoutes(router) = %t, want %t", got, tt.loadRoutes)
			}
			if got := g.shouldLoadRoutes(meshName, "other-router"); got != tt.loadOtherRoutes {
				t.Fatalf("shouldLoadRoutes(other router) = %t, want %t", got, tt.loadOtherRoutes)
			}
			if got := g.shouldLoadVirtualGateways(meshName); got != tt.loadGateways {
				t.Fatalf("shouldLoadVirtualGateways(mesh) = %t, want %t", got, tt.loadGateways)
			}
			if got := g.shouldLoadGatewayRoutes(meshName, "gateway"); got != tt.loadGatewayRoutes {
				t.Fatalf("shouldLoadGatewayRoutes(gateway) = %t, want %t", got, tt.loadGatewayRoutes)
			}
			if got := g.shouldLoadGatewayRoutes(meshName, "other-gateway"); got != tt.loadOtherGatewayRoute {
				t.Fatalf("shouldLoadGatewayRoutes(other gateway) = %t, want %t", got, tt.loadOtherGatewayRoute)
			}
			if got := g.shouldAppendMeshChildResource(appMeshVirtualNodeResourceType, node); got != tt.appendNode {
				t.Fatalf("shouldAppendMeshChildResource(node) = %t, want %t", got, tt.appendNode)
			}
			if got := g.shouldAppendMeshChildResource(appMeshVirtualRouterResourceType, router); got != tt.appendRouter {
				t.Fatalf("shouldAppendMeshChildResource(router) = %t, want %t", got, tt.appendRouter)
			}
			if got := g.shouldAppendMeshChildResource(appMeshRouteResourceType, route); got != tt.appendRoute {
				t.Fatalf("shouldAppendMeshChildResource(route) = %t, want %t", got, tt.appendRoute)
			}
			if got := g.shouldAppendMeshChildResource(appMeshVirtualGatewayResourceType, gateway); got != tt.appendGateway {
				t.Fatalf("shouldAppendMeshChildResource(gateway) = %t, want %t", got, tt.appendGateway)
			}
			if got := g.shouldAppendMeshChildResource(appMeshGatewayRouteResourceType, gatewayRoute); got != tt.appendGatewayRoute {
				t.Fatalf("shouldAppendMeshChildResource(gateway route) = %t, want %t", got, tt.appendGatewayRoute)
			}
		})
	}
}
