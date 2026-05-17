// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode"

	"github.com/IBM/sarama"
	"github.com/chenrui333/terraformer/terraformutils"
)

type ACLGenerator struct {
	Service
}

type ACL struct {
	Principal                 string
	Host                      string
	Operation                 string
	PermissionType            string
	ResourceType              string
	ResourceName              string
	ResourcePatternTypeFilter string
}

var ACLAllowEmptyValues = []string{}

func (g *ACLGenerator) InitResources() error {
	config := g.Args["config"].(Config)
	admin, err := g.admin(config)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := admin.Close(); closeErr != nil {
			log.Printf("kafka: close admin client: %v", closeErr)
		}
	}()

	acls, err := g.listACLs(admin)
	if err != nil {
		return err
	}
	g.Resources = g.createResources(acls)
	return nil
}

func (g *ACLGenerator) ParseFilter(rawFilter string) []terraformutils.ResourceFilter {
	for _, prefix := range []string{"kafka_acl=", "acls=", "acl="} {
		if strings.HasPrefix(rawFilter, prefix) {
			return []terraformutils.ResourceFilter{aclIDFilter(strings.TrimPrefix(rawFilter, prefix))}
		}
	}
	if filter, ok := parseACLIDFilter(rawFilter); ok {
		return []terraformutils.ResourceFilter{filter}
	}
	return g.Service.ParseFilter(rawFilter)
}

func (g *ACLGenerator) ParseFilters(rawFilters []string) {
	g.Filter = []terraformutils.ResourceFilter{}
	for _, rawFilter := range rawFilters {
		g.Filter = append(g.Filter, g.ParseFilter(rawFilter)...)
	}
}

func parseACLIDFilter(rawFilter string) (terraformutils.ResourceFilter, bool) {
	for _, prefix := range []string{"Type=acl;Name=id;Value=", "Name=id;Value="} {
		if strings.HasPrefix(rawFilter, prefix) {
			return aclIDFilter(strings.TrimPrefix(rawFilter, prefix)), true
		}
	}
	return terraformutils.ResourceFilter{}, false
}

func aclIDFilter(id string) terraformutils.ResourceFilter {
	return terraformutils.ResourceFilter{
		ServiceName:      "acl",
		FieldPath:        "id",
		AcceptableValues: []string{id},
	}
}

func (g *ACLGenerator) listACLs(admin adminClient) ([]ACL, error) {
	explicitACLs, err := g.explicitlyRequestedACLs()
	if err != nil {
		return nil, err
	}
	if len(explicitACLs) > 0 {
		return g.listExplicitACLs(admin, explicitACLs)
	}
	return g.listAllACLs(admin)
}

func (g *ACLGenerator) explicitlyRequestedACLs() ([]ACL, error) {
	seen := map[string]struct{}{}
	acls := []ACL{}
	for _, filter := range g.Filter {
		if filter.FieldPath != "id" || !filter.IsApplicable("acl") {
			continue
		}
		for _, value := range filter.AcceptableValues {
			acl, err := parseKafkaACLImportID(value)
			if err != nil {
				return nil, err
			}
			id := acl.ImportID()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			acls = append(acls, acl)
		}
	}
	sortACLs(acls)
	return acls, nil
}

func (g *ACLGenerator) listExplicitACLs(admin adminClient, requested []ACL) ([]ACL, error) {
	acls := []ACL{}
	for _, acl := range requested {
		filter, err := acl.SaramaFilter()
		if err != nil {
			return nil, err
		}
		resourceACLs, err := admin.ListAcls(filter)
		if err != nil {
			return nil, err
		}
		for _, found := range aclsFromResourceACLs(resourceACLs) {
			if found.ImportID() == acl.ImportID() {
				acls = append(acls, found)
			}
		}
	}
	return uniqueSortedACLs(acls), nil
}

func (g *ACLGenerator) listAllACLs(admin adminClient) ([]ACL, error) {
	resourceTypes := []sarama.AclResourceType{
		sarama.AclResourceTopic,
		sarama.AclResourceGroup,
		sarama.AclResourceCluster,
		sarama.AclResourceTransactionalID,
		sarama.AclResourceDelegationToken,
	}
	acls := []ACL{}
	for _, resourceType := range resourceTypes {
		resourceACLs, err := admin.ListAcls(sarama.AclFilter{
			ResourceType:              resourceType,
			ResourcePatternTypeFilter: sarama.AclPatternAny,
			Operation:                 sarama.AclOperationAny,
			PermissionType:            sarama.AclPermissionAny,
		})
		if err != nil {
			return nil, err
		}
		acls = append(acls, aclsFromResourceACLs(resourceACLs)...)
	}
	return uniqueSortedACLs(acls), nil
}

func aclsFromResourceACLs(resourceACLs []sarama.ResourceAcls) []ACL {
	acls := []ACL{}
	for _, resourceACL := range resourceACLs {
		for _, saramaACL := range resourceACL.Acls {
			if saramaACL == nil {
				continue
			}
			acl := ACL{
				Principal:                 saramaACL.Principal,
				Host:                      saramaACL.Host,
				Operation:                 aclOperationToString(saramaACL.Operation),
				PermissionType:            aclPermissionTypeToString(saramaACL.PermissionType),
				ResourceType:              aclResourceTypeToString(resourceACL.ResourceType),
				ResourceName:              resourceACL.ResourceName,
				ResourcePatternTypeFilter: aclResourcePatternTypeToString(resourceACL.ResourcePatternType),
			}
			if !acl.hasProviderSupportedPatternType() {
				log.Printf("kafka: skipping ACL %q because resource_pattern_type_filter %q is not supported by kafka_acl", acl.ImportID(), acl.ResourcePatternTypeFilter)
				continue
			}
			if !acl.isPipeDelimitedImportable() {
				log.Printf("kafka: skipping ACL for resource %q because kafka_acl import IDs cannot escape pipe characters", acl.ResourceName)
				continue
			}
			acls = append(acls, acl)
		}
	}
	return acls
}

func (g ACLGenerator) createResources(acls []ACL) []terraformutils.Resource {
	resources := make([]terraformutils.Resource, 0, len(acls))
	for _, acl := range acls {
		attributes := acl.attributes()
		additionalFields := map[string]interface{}{}
		for key, value := range attributes {
			additionalFields[key] = value
		}
		resources = append(resources, terraformutils.NewResource(
			acl.ImportID(),
			kafkaACLResourceName(acl),
			"kafka_acl",
			"kafka",
			attributes,
			ACLAllowEmptyValues,
			additionalFields,
		))
	}
	return resources
}

func (a ACL) attributes() map[string]string {
	return map[string]string{
		"acl_principal":                a.Principal,
		"acl_host":                     a.Host,
		"acl_operation":                a.Operation,
		"acl_permission_type":          a.PermissionType,
		"resource_type":                a.ResourceType,
		"resource_name":                a.ResourceName,
		"resource_pattern_type_filter": a.ResourcePatternTypeFilter,
	}
}

func (a ACL) ImportID() string {
	return strings.Join([]string{
		a.Principal,
		a.Host,
		a.Operation,
		a.PermissionType,
		a.ResourceType,
		a.ResourceName,
		a.ResourcePatternTypeFilter,
	}, "|")
}

func parseKafkaACLImportID(id string) (ACL, error) {
	parts := strings.Split(id, "|")
	if len(parts) != 7 {
		return ACL{}, fmt.Errorf("kafka acl import ID must have 7 pipe-delimited segments, got %d", len(parts))
	}
	return ACL{
		Principal:                 parts[0],
		Host:                      parts[1],
		Operation:                 parts[2],
		PermissionType:            parts[3],
		ResourceType:              parts[4],
		ResourceName:              parts[5],
		ResourcePatternTypeFilter: parts[6],
	}, nil
}

func (a ACL) SaramaFilter() (sarama.AclFilter, error) {
	resourceType, err := aclResourceTypeFromString(a.ResourceType)
	if err != nil {
		return sarama.AclFilter{}, err
	}
	patternType, err := aclResourcePatternTypeFromString(a.ResourcePatternTypeFilter)
	if err != nil {
		return sarama.AclFilter{}, err
	}
	operation, err := aclOperationFromString(a.Operation)
	if err != nil {
		return sarama.AclFilter{}, err
	}
	permissionType, err := aclPermissionTypeFromString(a.PermissionType)
	if err != nil {
		return sarama.AclFilter{}, err
	}
	principal := a.Principal
	host := a.Host
	resourceName := a.ResourceName
	return sarama.AclFilter{
		ResourceType:              resourceType,
		ResourceName:              &resourceName,
		ResourcePatternTypeFilter: patternType,
		Principal:                 &principal,
		Host:                      &host,
		Operation:                 operation,
		PermissionType:            permissionType,
	}, nil
}

func (a ACL) hasProviderSupportedPatternType() bool {
	return a.ResourcePatternTypeFilter == "Literal" || a.ResourcePatternTypeFilter == "Prefixed"
}

func (a ACL) isPipeDelimitedImportable() bool {
	for _, part := range []string{
		a.Principal,
		a.Host,
		a.Operation,
		a.PermissionType,
		a.ResourceType,
		a.ResourceName,
		a.ResourcePatternTypeFilter,
	} {
		if strings.Contains(part, "|") {
			return false
		}
	}
	return true
}

func kafkaACLResourceName(acl ACL) string {
	id := acl.ImportID()
	hash := sha256.Sum256([]byte(id))
	return "acl_" + normalizeACLResourceName(id) + "_" + hex.EncodeToString(hash[:4])
}

func normalizeACLResourceName(value string) string {
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			builder.WriteRune(r)
		case unicode.IsSpace(r):
			builder.WriteByte('_')
		default:
			fmt.Fprintf(&builder, "_x%04X_", r)
		}
	}
	name := strings.Trim(builder.String(), "_")
	if name == "" {
		return "acl"
	}
	return name
}

func uniqueSortedACLs(acls []ACL) []ACL {
	seen := map[string]struct{}{}
	unique := make([]ACL, 0, len(acls))
	for _, acl := range acls {
		id := acl.ImportID()
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, acl)
	}
	sortACLs(unique)
	return unique
}

func sortACLs(acls []ACL) {
	sort.Slice(acls, func(i, j int) bool {
		return acls[i].ImportID() < acls[j].ImportID()
	})
}

func aclOperationToString(in sarama.AclOperation) string {
	switch in {
	case sarama.AclOperationUnknown:
		return "Unknown"
	case sarama.AclOperationAny:
		return "Any"
	case sarama.AclOperationAll:
		return "All"
	case sarama.AclOperationRead:
		return "Read"
	case sarama.AclOperationWrite:
		return "Write"
	case sarama.AclOperationCreate:
		return "Create"
	case sarama.AclOperationDelete:
		return "Delete"
	case sarama.AclOperationAlter:
		return "Alter"
	case sarama.AclOperationDescribe:
		return "Describe"
	case sarama.AclOperationClusterAction:
		return "ClusterAction"
	case sarama.AclOperationDescribeConfigs:
		return "DescribeConfigs"
	case sarama.AclOperationAlterConfigs:
		return "AlterConfigs"
	case sarama.AclOperationIdempotentWrite:
		return "IdempotentWrite"
	}
	return "Unknown"
}

func aclOperationFromString(in string) (sarama.AclOperation, error) {
	switch in {
	case "Unknown":
		return sarama.AclOperationUnknown, nil
	case "Any":
		return sarama.AclOperationAny, nil
	case "All":
		return sarama.AclOperationAll, nil
	case "Read":
		return sarama.AclOperationRead, nil
	case "Write":
		return sarama.AclOperationWrite, nil
	case "Create":
		return sarama.AclOperationCreate, nil
	case "Delete":
		return sarama.AclOperationDelete, nil
	case "Alter":
		return sarama.AclOperationAlter, nil
	case "Describe":
		return sarama.AclOperationDescribe, nil
	case "ClusterAction":
		return sarama.AclOperationClusterAction, nil
	case "DescribeConfigs":
		return sarama.AclOperationDescribeConfigs, nil
	case "AlterConfigs":
		return sarama.AclOperationAlterConfigs, nil
	case "IdempotentWrite":
		return sarama.AclOperationIdempotentWrite, nil
	}
	return sarama.AclOperationUnknown, fmt.Errorf("kafka acl: unknown operation %q", in)
}

func aclPermissionTypeToString(in sarama.AclPermissionType) string {
	switch in {
	case sarama.AclPermissionUnknown:
		return "Unknown"
	case sarama.AclPermissionAny:
		return "Any"
	case sarama.AclPermissionDeny:
		return "Deny"
	case sarama.AclPermissionAllow:
		return "Allow"
	}
	return "Unknown"
}

func aclPermissionTypeFromString(in string) (sarama.AclPermissionType, error) {
	switch in {
	case "Unknown":
		return sarama.AclPermissionUnknown, nil
	case "Any":
		return sarama.AclPermissionAny, nil
	case "Deny":
		return sarama.AclPermissionDeny, nil
	case "Allow":
		return sarama.AclPermissionAllow, nil
	}
	return sarama.AclPermissionUnknown, fmt.Errorf("kafka acl: unknown permission type %q", in)
}

func aclResourceTypeToString(in sarama.AclResourceType) string {
	switch in {
	case sarama.AclResourceUnknown:
		return "Unknown"
	case sarama.AclResourceAny:
		return "Any"
	case sarama.AclResourceTopic:
		return "Topic"
	case sarama.AclResourceGroup:
		return "Group"
	case sarama.AclResourceCluster:
		return "Cluster"
	case sarama.AclResourceTransactionalID:
		return "TransactionalID"
	case sarama.AclResourceDelegationToken:
		return "DelegationToken"
	}
	return "Unknown"
}

func aclResourceTypeFromString(in string) (sarama.AclResourceType, error) {
	switch in {
	case "Unknown":
		return sarama.AclResourceUnknown, nil
	case "Any":
		return sarama.AclResourceAny, nil
	case "Topic":
		return sarama.AclResourceTopic, nil
	case "Group":
		return sarama.AclResourceGroup, nil
	case "Cluster":
		return sarama.AclResourceCluster, nil
	case "TransactionalID":
		return sarama.AclResourceTransactionalID, nil
	case "DelegationToken":
		return sarama.AclResourceDelegationToken, nil
	}
	return sarama.AclResourceUnknown, fmt.Errorf("kafka acl: unknown resource type %q", in)
}

func aclResourcePatternTypeToString(in sarama.AclResourcePatternType) string {
	switch in {
	case sarama.AclPatternUnknown:
		return "Unknown"
	case sarama.AclPatternAny:
		return "Any"
	case sarama.AclPatternMatch:
		return "Match"
	case sarama.AclPatternLiteral:
		return "Literal"
	case sarama.AclPatternPrefixed:
		return "Prefixed"
	}
	return "Unknown"
}

func aclResourcePatternTypeFromString(in string) (sarama.AclResourcePatternType, error) {
	switch in {
	case "Unknown":
		return sarama.AclPatternUnknown, nil
	case "Any":
		return sarama.AclPatternAny, nil
	case "Match":
		return sarama.AclPatternMatch, nil
	case "Literal":
		return sarama.AclPatternLiteral, nil
	case "Prefixed":
		return sarama.AclPatternPrefixed, nil
	}
	return sarama.AclPatternUnknown, fmt.Errorf("kafka acl: unknown resource pattern type filter %q", in)
}
