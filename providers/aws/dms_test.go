// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"errors"
	"testing"

	dmstypes "github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/aws/smithy-go"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestDMSResourceName(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "filters empty parts", parts: []string{"", "endpoint", "", "orders"}, want: "endpoint/orders"},
		{name: "preserves segment boundaries", parts: []string{"replication-task", "orders"}, want: "replication-task/orders"},
		{name: "fallback", parts: nil, want: "dms-resource"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dmsResourceName(tt.parts...)
			if got != tt.want {
				t.Fatalf("dmsResourceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDMSResourceNamesAvoidTypeCollisions(t *testing.T) {
	endpoint := dmsResourceName("endpoint", "shared")
	s3Endpoint := dmsResourceName("s3-endpoint", "shared")
	task := dmsResourceName("replication-task", "shared")
	if endpoint == task {
		t.Fatalf("resource names collide: %q", endpoint)
	}
	if endpoint == s3Endpoint {
		t.Fatalf("endpoint resource names collide: %q", endpoint)
	}
}

func TestDMSOptionalResourceLoaderErrors(t *testing.T) {
	t.Run("skips resource not found", func(t *testing.T) {
		g := DmsGenerator{}
		calledNext := false
		err := g.loadOptionalResources([]dmsOptionalResourceLoader{
			{
				name: "certificates",
				load: func() error {
					return &dmstypes.ResourceNotFoundFault{}
				},
			},
			{
				name: "event subscriptions",
				load: func() error {
					calledNext = true
					return nil
				},
			},
		})
		if err != nil {
			t.Fatalf("loadOptionalResources() error = %v, want nil", err)
		}
		if !calledNext {
			t.Fatal("loadOptionalResources() should continue after skippable error")
		}
	})

	t.Run("skips access denied", func(t *testing.T) {
		g := DmsGenerator{}
		calledNext := false
		err := g.loadOptionalResources([]dmsOptionalResourceLoader{
			{
				name: "replication configs",
				load: func() error {
					return &dmstypes.AccessDeniedFault{}
				},
			},
			{
				name: "event subscriptions",
				load: func() error {
					calledNext = true
					return nil
				},
			},
		})
		if err != nil {
			t.Fatalf("loadOptionalResources() error = %v, want nil", err)
		}
		if !calledNext {
			t.Fatal("loadOptionalResources() should continue after access denied")
		}
	})

	t.Run("returns unexpected error", func(t *testing.T) {
		g := DmsGenerator{}
		boom := errors.New("boom")
		calledNext := false
		err := g.loadOptionalResources([]dmsOptionalResourceLoader{
			{
				name: "certificates",
				load: func() error {
					return boom
				},
			},
			{
				name: "event subscriptions",
				load: func() error {
					calledNext = true
					return nil
				},
			},
		})
		if !errors.Is(err, boom) {
			t.Fatalf("loadOptionalResources() error = %v, want %v", err, boom)
		}
		if calledNext {
			t.Fatal("loadOptionalResources() should stop after unexpected error")
		}
	})
}

func TestDMSOptionalResourceErrorSkippable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "resource not found", err: &dmstypes.ResourceNotFoundFault{}, want: true},
		{name: "typed access denied", err: &dmstypes.AccessDeniedFault{}, want: true},
		{name: "generic access denied", err: &smithy.GenericAPIError{Code: "AccessDeniedException"}, want: true},
		{name: "generic access denied fault", err: &smithy.GenericAPIError{Code: "AccessDeniedFault"}, want: true},
		{name: "unexpected generic error", err: &smithy.GenericAPIError{Code: "ThrottlingException"}, want: false},
		{name: "plain error", err: errors.New("boom"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dmsOptionalResourceErrorSkippable(tt.err); got != tt.want {
				t.Fatalf("dmsOptionalResourceErrorSkippable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDMSCertificateResource(t *testing.T) {
	resource, ok := newDMSCertificateResource(dmstypes.Certificate{
		CertificateIdentifier: dmsString("dms-cert"),
		CertificatePem:        dmsString("-----BEGIN CERTIFICATE-----"),
	})
	assertDMSResource(t, resource, ok, "dms-cert", dmsResourceName("certificate", "dms-cert"), dmsCertificateResourceType)

	if _, ok := newDMSCertificateResource(dmstypes.Certificate{CertificatePem: dmsString("-----BEGIN CERTIFICATE-----")}); ok {
		t.Fatal("certificate with empty identifier should be skipped")
	}
	if _, ok := newDMSCertificateResource(dmstypes.Certificate{CertificateIdentifier: dmsString("dms-cert")}); ok {
		t.Fatal("certificate without PEM or wallet material should be skipped")
	}
}

func TestNewDMSEventSubscriptionResource(t *testing.T) {
	resource, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		Enabled:             true,
		EventCategoriesList: []string{"creation", "failure"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "dms-events", dmsResourceName("event-subscription", "dms-events"), dmsEventSubscriptionResourceType)
	if got := resource.InstanceState.Attributes["event_categories.#"]; got != "2" {
		t.Fatalf("event_categories.# = %q, want 2", got)
	}
	if got := resource.InstanceState.Attributes["event_categories.1"]; got != "failure" {
		t.Fatalf("event_categories.1 = %q, want failure", got)
	}
	if _, ok := resource.AdditionalFields["event_categories"]; ok {
		t.Fatal("non-empty event categories should be read from refreshed state")
	}
	if _, ok := resource.InstanceState.Attributes["enabled"]; ok {
		t.Fatal("enabled=true event subscription should use the provider default")
	}

	resource, ok = newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId: dmsString("dms-all-events"),
		Enabled:            true,
		SnsTopicArn:        dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:         dmsString("replication-instance"),
		Status:             dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "dms-all-events", dmsResourceName("event-subscription", "dms-all-events"), dmsEventSubscriptionResourceType)
	if got := resource.InstanceState.Attributes["event_categories.#"]; got != "0" {
		t.Fatalf("all-category event_categories.# = %q, want 0", got)
	}
	emptyCategories, ok := resource.AdditionalFields["event_categories"].([]interface{})
	if !ok {
		t.Fatalf("all-category event_categories additional field type = %T, want []interface{}", resource.AdditionalFields["event_categories"])
	}
	if len(emptyCategories) != 0 {
		t.Fatalf("all-category event_categories length = %d, want 0", len(emptyCategories))
	}

	resource, ok = newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-disabled-events"),
		Enabled:             false,
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "dms-disabled-events", dmsResourceName("event-subscription", "dms-disabled-events"), dmsEventSubscriptionResourceType)
	if got := resource.InstanceState.Attributes["enabled"]; got != "false" {
		t.Fatalf("disabled event subscription enabled = %q, want false", got)
	}
	assertDMSAllowEmptyValue(t, resource, "enabled")

	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("active"),
	}); ok {
		t.Fatal("event subscription with empty name should be skipped")
	}
	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-task"),
		Status:              dmsString("creating"),
	}); ok {
		t.Fatal("creating event subscription should be skipped")
	}
	if _, ok := newDMSEventSubscriptionResource(dmstypes.EventSubscription{
		CustSubscriptionId:  dmsString("dms-events"),
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("security-group"),
		Status:              dmsString("active"),
	}); ok {
		t.Fatal("event subscription with unsupported source type should be skipped")
	}
}

func TestNewDMSReplicationConfigResource(t *testing.T) {
	multiAZ := false
	resource, ok := newDMSReplicationConfigResource(dmstypes.ReplicationConfig{
		ComputeConfig: &dmstypes.ComputeConfig{
			AvailabilityZone:           dmsString("us-east-1a"),
			DnsNameServers:             dmsString("1.1.1.1,2.2.2.2"),
			KmsKeyId:                   dmsString("arn:aws:kms:us-east-1:123456789012:key/example"),
			MaxCapacityUnits:           dmsInt32(64),
			MinCapacityUnits:           dmsInt32(2),
			MultiAZ:                    &multiAZ,
			PreferredMaintenanceWindow: dmsString("sun:23:45-mon:00:30"),
			ReplicationSubnetGroupId:   dmsString("dms-subnets"),
			VpcSecurityGroupIds:        []string{"sg-1", "sg-2"},
		},
		ReplicationConfigArn:        dmsString("arn:aws:dms:us-east-1:123456789012:replication-config:ABC123"),
		ReplicationConfigIdentifier: dmsString("orders"),
		ReplicationSettings:         dmsString("{\"Logging\":{\"EnableLogging\":true}}"),
		ReplicationType:             dmstypes.MigrationTypeValueCdc,
		SourceEndpointArn:           dmsString("arn:aws:dms:us-east-1:123456789012:endpoint:SOURCE"),
		SupplementalSettings:        dmsString("{}"),
		TableMappings:               dmsString("{\"rules\":[]}"),
		TargetEndpointArn:           dmsString("arn:aws:dms:us-east-1:123456789012:endpoint:TARGET"),
	})
	assertDMSResource(
		t,
		resource,
		ok,
		"arn:aws:dms:us-east-1:123456789012:replication-config:ABC123",
		dmsResourceName("replication-config", "orders", "ABC123"),
		dmsReplicationConfigResourceType,
	)

	attributes := resource.InstanceState.Attributes
	expected := map[string]string{
		"compute_config.#":                              "1",
		"compute_config.0.availability_zone":            "us-east-1a",
		"compute_config.0.dns_name_servers":             "1.1.1.1,2.2.2.2",
		"compute_config.0.kms_key_id":                   "arn:aws:kms:us-east-1:123456789012:key/example",
		"compute_config.0.max_capacity_units":           "64",
		"compute_config.0.min_capacity_units":           "2",
		"compute_config.0.multi_az":                     "false",
		"compute_config.0.preferred_maintenance_window": "sun:23:45-mon:00:30",
		"compute_config.0.replication_subnet_group_id":  "dms-subnets",
		"compute_config.0.vpc_security_group_ids.#":     "2",
		"compute_config.0.vpc_security_group_ids.1":     "sg-2",
		"replication_config_identifier":                 "orders",
		"replication_settings":                          "{\"Logging\":{\"EnableLogging\":true}}",
		"replication_type":                              "cdc",
		"source_endpoint_arn":                           "arn:aws:dms:us-east-1:123456789012:endpoint:SOURCE",
		"supplemental_settings":                         "{}",
		"table_mappings":                                "{\"rules\":[]}",
		"target_endpoint_arn":                           "arn:aws:dms:us-east-1:123456789012:endpoint:TARGET",
	}
	for key, want := range expected {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestNewDMSReplicationConfigResourceSkipsUnsafeConfigs(t *testing.T) {
	validConfig := func() dmstypes.ReplicationConfig {
		return dmstypes.ReplicationConfig{
			ComputeConfig: &dmstypes.ComputeConfig{
				MaxCapacityUnits:         dmsInt32(64),
				ReplicationSubnetGroupId: dmsString("dms-subnets"),
			},
			ReplicationConfigArn:        dmsString("arn:aws:dms:us-east-1:123456789012:replication-config:ABC123"),
			ReplicationConfigIdentifier: dmsString("orders"),
			ReplicationType:             dmstypes.MigrationTypeValueCdc,
			SourceEndpointArn:           dmsString("arn:aws:dms:us-east-1:123456789012:endpoint:SOURCE"),
			TableMappings:               dmsString("{\"rules\":[]}"),
			TargetEndpointArn:           dmsString("arn:aws:dms:us-east-1:123456789012:endpoint:TARGET"),
		}
	}

	tests := []struct {
		name   string
		mutate func(*dmstypes.ReplicationConfig)
	}{
		{name: "empty ARN", mutate: func(config *dmstypes.ReplicationConfig) { config.ReplicationConfigArn = nil }},
		{name: "empty identifier", mutate: func(config *dmstypes.ReplicationConfig) { config.ReplicationConfigIdentifier = nil }},
		{name: "empty replication type", mutate: func(config *dmstypes.ReplicationConfig) { config.ReplicationType = "" }},
		{name: "empty source endpoint ARN", mutate: func(config *dmstypes.ReplicationConfig) { config.SourceEndpointArn = nil }},
		{name: "empty target endpoint ARN", mutate: func(config *dmstypes.ReplicationConfig) { config.TargetEndpointArn = nil }},
		{name: "empty table mappings", mutate: func(config *dmstypes.ReplicationConfig) { config.TableMappings = nil }},
		{name: "missing compute config", mutate: func(config *dmstypes.ReplicationConfig) { config.ComputeConfig = nil }},
		{name: "missing replication subnet group", mutate: func(config *dmstypes.ReplicationConfig) { config.ComputeConfig.ReplicationSubnetGroupId = nil }},
		{name: "missing max capacity", mutate: func(config *dmstypes.ReplicationConfig) { config.ComputeConfig.MaxCapacityUnits = nil }},
		{name: "read-only", mutate: func(config *dmstypes.ReplicationConfig) { config.IsReadOnly = dmsBool(true) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.mutate(&config)
			if _, ok := newDMSReplicationConfigResource(config); ok {
				t.Fatal("replication config should be skipped")
			}
		})
	}
}

func TestDMSReplicationConfigResourceNameAvoidsARNCollisions(t *testing.T) {
	first := dmsReplicationConfigResourceName(dmstypes.ReplicationConfig{
		ReplicationConfigArn:        dmsString("arn:aws:dms:us-east-1:123456789012:replication-config:ABC123"),
		ReplicationConfigIdentifier: dmsString("orders"),
	})
	second := dmsReplicationConfigResourceName(dmstypes.ReplicationConfig{
		ReplicationConfigArn:        dmsString("arn:aws:dms:us-east-1:123456789012:replication-config:XYZ789"),
		ReplicationConfigIdentifier: dmsString("orders"),
	})
	if first == second {
		t.Fatalf("replication config resource names collide: %q", first)
	}
}

func TestNewDMSReplicationInstanceResource(t *testing.T) {
	resource, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{
		ReplicationInstanceIdentifier: dmsString("dms-ri"),
		ReplicationInstanceStatus:     dmsString("available"),
	})
	assertDMSResource(t, resource, ok, "dms-ri", dmsResourceName("replication-instance", "dms-ri"), dmsReplicationInstanceResourceType)

	if _, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString("available")}); ok {
		t.Fatal("replication instance with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationInstanceResource(dmstypes.ReplicationInstance{
		ReplicationInstanceIdentifier: dmsString("dms-ri"),
		ReplicationInstanceStatus:     dmsString("creating"),
	}); ok {
		t.Fatal("creating replication instance should be skipped")
	}
}

func TestNewDMSReplicationSubnetGroupResource(t *testing.T) {
	resource, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{
		ReplicationSubnetGroupIdentifier: dmsString("dms-subnets"),
		SubnetGroupStatus:                dmsString("Complete"),
	})
	assertDMSResource(t, resource, ok, "dms-subnets", dmsResourceName("replication-subnet-group", "dms-subnets"), dmsReplicationSubnetGroupResourceType)

	if _, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{SubnetGroupStatus: dmsString("Complete")}); ok {
		t.Fatal("replication subnet group with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationSubnetGroupResource(dmstypes.ReplicationSubnetGroup{
		ReplicationSubnetGroupIdentifier: dmsString("dms-subnets"),
		SubnetGroupStatus:                dmsString("deleting"),
	}); ok {
		t.Fatal("deleting replication subnet group should be skipped")
	}
}

func TestNewDMSEndpointResource(t *testing.T) {
	resource, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("source-endpoint"),
		EngineName:         dmsString("postgres"),
		Status:             dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "source-endpoint", dmsResourceName("endpoint", "source-endpoint"), dmsEndpointResourceType)

	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{Status: dmsString("active")}); ok {
		t.Fatal("endpoint with empty identifier should be skipped")
	}
	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("s3-target"),
		EndpointType:       dmstypes.ReplicationEndpointTypeValueTarget,
		EngineName:         dmsString("s3"),
		S3Settings: &dmstypes.S3Settings{
			BucketName:           dmsString("dms-export-bucket"),
			ServiceAccessRoleArn: dmsString("arn:aws:iam::123456789012:role/dms-s3-role"),
		},
		Status: dmsString("active"),
	}); ok {
		t.Fatal("S3 endpoint should be skipped for aws_dms_endpoint")
	}
	if _, ok := newDMSEndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("source-endpoint"),
		EngineName:         dmsString("postgres"),
		Status:             dmsString("failed"),
	}); ok {
		t.Fatal("failed endpoint should be skipped")
	}
}

func TestNewDMSS3EndpointResource(t *testing.T) {
	resource, ok := newDMSS3EndpointResource(dmstypes.Endpoint{
		CertificateArn:     dmsString("arn:aws:dms:us-east-1:123456789012:cert:ABC123"),
		EndpointIdentifier: dmsString("s3-target"),
		EndpointType:       dmstypes.ReplicationEndpointTypeValueTarget,
		EngineName:         dmsString("S3"),
		KmsKeyId:           dmsString("arn:aws:kms:us-east-1:123456789012:key/example"),
		S3Settings: &dmstypes.S3Settings{
			BucketFolder:         dmsString("exports"),
			BucketName:           dmsString("dms-export-bucket"),
			EnableStatistics:     dmsBool(false),
			Rfc4180:              dmsBool(false),
			ServiceAccessRoleArn: dmsString("arn:aws:iam::123456789012:role/dms-s3-role"),
		},
		SslMode: dmstypes.DmsSslModeValueNone,
		Status:  dmsString("active"),
	})
	assertDMSResource(t, resource, ok, "s3-target", dmsResourceName("s3-endpoint", "s3-target"), dmsS3EndpointResourceType)
	attributes := resource.InstanceState.Attributes
	expected := map[string]string{
		"bucket_folder":           "exports",
		"bucket_name":             "dms-export-bucket",
		"certificate_arn":         "arn:aws:dms:us-east-1:123456789012:cert:ABC123",
		"enable_statistics":       "false",
		"endpoint_id":             "s3-target",
		"endpoint_type":           "target",
		"kms_key_arn":             "arn:aws:kms:us-east-1:123456789012:key/example",
		"rfc_4180":                "false",
		"service_access_role_arn": "arn:aws:iam::123456789012:role/dms-s3-role",
		"ssl_mode":                "none",
	}
	for key, want := range expected {
		if got := attributes[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	assertDMSAllowEmptyValue(t, resource, "enable_statistics")
	assertDMSAllowEmptyValue(t, resource, "rfc_4180")

	sourceResource, ok := newDMSS3EndpointResource(dmstypes.Endpoint{
		EndpointIdentifier: dmsString("s3-source"),
		EndpointType:       dmstypes.ReplicationEndpointTypeValueSource,
		EngineName:         dmsString("s3"),
		S3Settings: &dmstypes.S3Settings{
			BucketName:              dmsString("dms-source-bucket"),
			ExternalTableDefinition: dmsString("{\"TableCount\":1}"),
			ServiceAccessRoleArn:    dmsString("arn:aws:iam::123456789012:role/dms-s3-role"),
		},
		Status: dmsString("active"),
	})
	assertDMSResource(t, sourceResource, ok, "s3-source", dmsResourceName("s3-endpoint", "s3-source"), dmsS3EndpointResourceType)
	if got := sourceResource.InstanceState.Attributes["external_table_definition"]; got != "{\"TableCount\":1}" {
		t.Fatalf("external_table_definition = %q, want source table definition", got)
	}
}

func TestNewDMSS3EndpointResourceSkipsUnsafeEndpoints(t *testing.T) {
	validEndpoint := func() dmstypes.Endpoint {
		return dmstypes.Endpoint{
			EndpointIdentifier: dmsString("s3-target"),
			EndpointType:       dmstypes.ReplicationEndpointTypeValueTarget,
			EngineName:         dmsString("s3"),
			S3Settings: &dmstypes.S3Settings{
				BucketName:           dmsString("dms-export-bucket"),
				ServiceAccessRoleArn: dmsString("arn:aws:iam::123456789012:role/dms-s3-role"),
			},
			Status: dmsString("active"),
		}
	}

	tests := []struct {
		name   string
		mutate func(*dmstypes.Endpoint)
	}{
		{name: "empty identifier", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.EndpointIdentifier = nil }},
		{name: "non-S3 engine", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.EngineName = dmsString("postgres") }},
		{name: "empty endpoint type", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.EndpointType = "" }},
		{name: "creating status", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.Status = dmsString("creating") }},
		{name: "missing S3 settings", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.S3Settings = nil }},
		{name: "missing bucket", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.S3Settings.BucketName = nil }},
		{name: "missing service role", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.S3Settings.ServiceAccessRoleArn = nil }},
		{name: "read-only", mutate: func(endpoint *dmstypes.Endpoint) { endpoint.IsReadOnly = dmsBool(true) }},
		{name: "source missing external table definition", mutate: func(endpoint *dmstypes.Endpoint) {
			endpoint.EndpointType = dmstypes.ReplicationEndpointTypeValueSource
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := validEndpoint()
			tt.mutate(&endpoint)
			if _, ok := newDMSS3EndpointResource(endpoint); ok {
				t.Fatal("S3 endpoint should be skipped")
			}
		})
	}
}

func TestNewDMSReplicationTaskResource(t *testing.T) {
	resource, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{
		ReplicationTaskIdentifier: dmsString("orders-task"),
		Status:                    dmsString("ready"),
	})
	assertDMSResource(t, resource, ok, "orders-task", dmsResourceName("replication-task", "orders-task"), dmsReplicationTaskResourceType)

	if _, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{Status: dmsString("ready")}); ok {
		t.Fatal("replication task with empty identifier should be skipped")
	}
	if _, ok := newDMSReplicationTaskResource(dmstypes.ReplicationTask{
		ReplicationTaskIdentifier: dmsString("orders-task"),
		Status:                    dmsString("creating"),
	}); ok {
		t.Fatal("creating replication task should be skipped")
	}
}

func TestDMSCertificateImportable(t *testing.T) {
	if !dmsCertificateImportable(dmstypes.Certificate{CertificatePem: dmsString("pem")}) {
		t.Fatal("certificate with PEM material should be importable")
	}
	if !dmsCertificateImportable(dmstypes.Certificate{CertificateWallet: []byte("wallet")}) {
		t.Fatal("certificate with wallet material should be importable")
	}
	if dmsCertificateImportable(dmstypes.Certificate{}) {
		t.Fatal("certificate without PEM or wallet material should not be importable")
	}
}

func TestDMSStringSliceAttributes(t *testing.T) {
	attributes := dmsStringSliceAttributes("event_categories", []string{"creation", "failure"})
	if got := attributes["event_categories.#"]; got != "2" {
		t.Fatalf("event_categories.# = %q, want 2", got)
	}
	if got := attributes["event_categories.0"]; got != "creation" {
		t.Fatalf("event_categories.0 = %q, want creation", got)
	}
	if got := attributes["event_categories.1"]; got != "failure" {
		t.Fatalf("event_categories.1 = %q, want failure", got)
	}

	emptyAttributes := dmsStringSliceAttributes("event_categories", nil)
	if got := emptyAttributes["event_categories.#"]; got != "0" {
		t.Fatalf("empty event_categories.# = %q, want 0", got)
	}
	if len(emptyAttributes) != 1 {
		t.Fatalf("empty attributes length = %d, want 1", len(emptyAttributes))
	}
}

func TestDMSEventSubscriptionImportable(t *testing.T) {
	base := dmstypes.EventSubscription{
		EventCategoriesList: []string{"creation"},
		SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
		SourceType:          dmsString("replication-instance"),
		Status:              dmsString("active"),
	}
	if !dmsEventSubscriptionImportable(base) {
		t.Fatal("active event subscription with supported source type and required fields should be importable")
	}
	allCategories := base
	allCategories.EventCategoriesList = nil
	if !dmsEventSubscriptionImportable(allCategories) {
		t.Fatal("active event subscription without event categories should import all categories")
	}
	caseVariant := base
	caseVariant.SnsTopicArn = dmsString("arn:aws:sns:us-east-1:123456789012:DmS-EvEnTs")
	caseVariant.SourceType = dmsString("RePlIcAtIoN-InStAnCe")
	caseVariant.Status = dmsString("AcTiVe")
	if !dmsEventSubscriptionImportable(caseVariant) {
		t.Fatal("event subscription importability should be case-insensitive for status and source type")
	}

	tests := []struct {
		name         string
		subscription dmstypes.EventSubscription
	}{
		{name: "empty status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
		}},
		{name: "creating status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("creating"),
		}},
		{name: "no permission status", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("no-permission"),
		}},
		{name: "unsupported source type", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SnsTopicArn:         dmsString("arn:aws:sns:us-east-1:123456789012:dms-events"),
			SourceType:          dmsString("security-group"),
			Status:              dmsString("active"),
		}},
		{name: "empty sns topic arn", subscription: dmstypes.EventSubscription{
			EventCategoriesList: []string{"creation"},
			SourceType:          dmsString("replication-instance"),
			Status:              dmsString("active"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if dmsEventSubscriptionImportable(tt.subscription) {
				t.Fatal("event subscription should not be importable")
			}
		})
	}
}

func TestDMSEventSubscriptionSourceTypeImportable(t *testing.T) {
	for _, sourceType := range []string{"replication-instance", "replication-task", "REPLICATION-INSTANCE", "RePlIcAtIoN-TaSk"} {
		if !dmsEventSubscriptionSourceTypeImportable(sourceType) {
			t.Fatalf("source type %q should be importable", sourceType)
		}
	}
	for _, sourceType := range []string{"", "replication-server", "security-group", "SeCuRiTy-GrOuP"} {
		if dmsEventSubscriptionSourceTypeImportable(sourceType) {
			t.Fatalf("source type %q should not be importable", sourceType)
		}
	}
}

func TestDMSReplicationInstanceImportable(t *testing.T) {
	if !dmsReplicationInstanceImportable(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString("available")}) {
		t.Fatal("available replication instance should be importable")
	}
	for _, status := range []string{"creating", "deleting", "failed", "modifying", "upgrading", ""} {
		if dmsReplicationInstanceImportable(dmstypes.ReplicationInstance{ReplicationInstanceStatus: dmsString(status)}) {
			t.Fatalf("replication instance status %q should not be importable", status)
		}
	}
}

func TestDMSEndpointImportable(t *testing.T) {
	if !dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres"), Status: dmsString("active")}) {
		t.Fatal("active non-S3 endpoint should be importable")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("s3"), Status: dmsString("active")}) {
		t.Fatal("S3 endpoint should not be imported as aws_dms_endpoint")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres"), Status: dmsString("failed")}) {
		t.Fatal("failed endpoint should not be importable")
	}
	if dmsEndpointImportable(dmstypes.Endpoint{EngineName: dmsString("postgres")}) {
		t.Fatal("endpoint with empty status should not be importable")
	}
}

func TestDMSReplicationTaskImportable(t *testing.T) {
	for _, status := range []string{"ready", "running", "stopped"} {
		if !dmsReplicationTaskImportable(dmstypes.ReplicationTask{Status: dmsString(status)}) {
			t.Fatalf("replication task status %q should be importable", status)
		}
	}
	for _, status := range []string{"creating", "deleting", "failed", "failed-move", "modifying", "moving", "starting", "stopping", ""} {
		if dmsReplicationTaskImportable(dmstypes.ReplicationTask{Status: dmsString(status)}) {
			t.Fatalf("replication task status %q should not be importable", status)
		}
	}
}

func TestDMSStableStatusImportable(t *testing.T) {
	for _, status := range []string{"active", "available", "Complete", "ready", "running", "stopped"} {
		if !dmsStableStatusImportable(status) {
			t.Fatalf("status %q should be importable", status)
		}
	}
	for _, status := range []string{"", "creating", "deleting", "failed", "failed-move", "modifying", "moving", "starting", "stopping", "testing", "upgrading"} {
		if dmsStableStatusImportable(status) {
			t.Fatalf("status %q should not be importable", status)
		}
	}
}

func assertDMSResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantName, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource was skipped")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.ResourceName; got != terraformutils.TfSanitize(wantName) {
		t.Fatalf("resource name = %q, want %q", got, terraformutils.TfSanitize(wantName))
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
}

func assertDMSAllowEmptyValue(t *testing.T, resource terraformutils.Resource, want string) {
	t.Helper()
	for _, value := range resource.AllowEmptyValues {
		if value == "^"+want+"$" {
			return
		}
	}
	t.Fatalf("AllowEmptyValues = %v, want anchored %q", resource.AllowEmptyValues, want)
}

func dmsString(value string) *string {
	return &value
}

func dmsBool(value bool) *bool {
	return &value
}

func dmsInt32(value int32) *int32 {
	return &value
}
