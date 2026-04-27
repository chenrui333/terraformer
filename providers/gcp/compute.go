// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

// Map of supported GCP compute service with code generate
var ComputeServices = map[string]terraformutils.ServiceGenerator{

	"addresses":                   &GCPFacade{service: &AddressesGenerator{}},
	"autoscalers":                 &GCPFacade{service: &AutoscalersGenerator{}},
	"backendBuckets":              &GCPFacade{service: &BackendBucketsGenerator{}},
	"backendServices":             &GCPFacade{service: &BackendServicesGenerator{}},
	"disks":                       &GCPFacade{service: &DisksGenerator{}},
	"externalVpnGateways":         &GCPFacade{service: &ExternalVpnGatewaysGenerator{}},
	"firewall":                    &GCPFacade{service: &FirewallGenerator{}},
	"forwardingRules":             &GCPFacade{service: &ForwardingRulesGenerator{}},
	"globalAddresses":             &GCPFacade{service: &GlobalAddressesGenerator{}},
	"globalForwardingRules":       &GCPFacade{service: &GlobalForwardingRulesGenerator{}},
	"healthChecks":                &GCPFacade{service: &HealthChecksGenerator{}},
	"httpHealthChecks":            &GCPFacade{service: &HttpHealthChecksGenerator{}},
	"httpsHealthChecks":           &GCPFacade{service: &HttpsHealthChecksGenerator{}},
	"images":                      &GCPFacade{service: &ImagesGenerator{}},
	"instanceGroupManagers":       &GCPFacade{service: &InstanceGroupManagersGenerator{}},
	"instanceGroups":              &GCPFacade{service: &InstanceGroupsGenerator{}},
	"instanceTemplates":           &GCPFacade{service: &InstanceTemplatesGenerator{}},
	"interconnectAttachments":     &GCPFacade{service: &InterconnectAttachmentsGenerator{}},
	"networkEndpointGroups":       &GCPFacade{service: &NetworkEndpointGroupsGenerator{}},
	"networks":                    &GCPFacade{service: &NetworksGenerator{}},
	"nodeGroups":                  &GCPFacade{service: &NodeGroupsGenerator{}},
	"nodeTemplates":               &GCPFacade{service: &NodeTemplatesGenerator{}},
	"packetMirrorings":            &GCPFacade{service: &PacketMirroringsGenerator{}},
	"regionAutoscalers":           &GCPFacade{service: &RegionAutoscalersGenerator{}},
	"regionBackendServices":       &GCPFacade{service: &RegionBackendServicesGenerator{}},
	"regionDisks":                 &GCPFacade{service: &RegionDisksGenerator{}},
	"regionHealthChecks":          &GCPFacade{service: &RegionHealthChecksGenerator{}},
	"regionInstanceGroupManagers": &GCPFacade{service: &RegionInstanceGroupManagersGenerator{}},
	"regionInstanceGroups":        &GCPFacade{service: &RegionInstanceGroupsGenerator{}},
	"regionSslCertificates":       &GCPFacade{service: &RegionSslCertificatesGenerator{}},
	"regionTargetHttpProxies":     &GCPFacade{service: &RegionTargetHttpProxiesGenerator{}},
	"regionTargetHttpsProxies":    &GCPFacade{service: &RegionTargetHttpsProxiesGenerator{}},
	"regionUrlMaps":               &GCPFacade{service: &RegionUrlMapsGenerator{}},
	"reservations":                &GCPFacade{service: &ReservationsGenerator{}},
	"resourcePolicies":            &GCPFacade{service: &ResourcePoliciesGenerator{}},
	"routers":                     &GCPFacade{service: &RoutersGenerator{}},
	"routes":                      &GCPFacade{service: &RoutesGenerator{}},
	"securityPolicies":            &GCPFacade{service: &SecurityPoliciesGenerator{}},
	"sslCertificates":             &GCPFacade{service: &SslCertificatesGenerator{}},
	"sslPolicies":                 &GCPFacade{service: &SslPoliciesGenerator{}},
	"subnetworks":                 &GCPFacade{service: &SubnetworksGenerator{}},
	"targetHttpProxies":           &GCPFacade{service: &TargetHttpProxiesGenerator{}},
	"targetHttpsProxies":          &GCPFacade{service: &TargetHttpsProxiesGenerator{}},
	"targetInstances":             &GCPFacade{service: &TargetInstancesGenerator{}},
	"targetPools":                 &GCPFacade{service: &TargetPoolsGenerator{}},
	"targetSslProxies":            &GCPFacade{service: &TargetSslProxiesGenerator{}},
	"targetTcpProxies":            &GCPFacade{service: &TargetTcpProxiesGenerator{}},
	"targetVpnGateways":           &GCPFacade{service: &TargetVpnGatewaysGenerator{}},
	"urlMaps":                     &GCPFacade{service: &UrlMapsGenerator{}},
	"vpnTunnels":                  &GCPFacade{service: &VpnTunnelsGenerator{}},
}
