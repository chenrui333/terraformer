// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/quicksight"
	quicksighttypes "github.com/aws/aws-sdk-go-v2/service/quicksight/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

func TestQuickSightImportIDs(t *testing.T) {
	if got, want := quickSightNamespaceImportID("123456789012", "default"), "123456789012,default"; got != want {
		t.Fatalf("quickSightNamespaceImportID() = %q, want %q", got, want)
	}
	if got, want := quickSightGroupImportID("123456789012", "default", "authors"), "123456789012/default/authors"; got != want {
		t.Fatalf("quickSightGroupImportID() = %q, want %q", got, want)
	}
	if got, want := quickSightGroupMembershipImportID("123456789012", "default", "authors", "alice"), "123456789012/default/authors/alice"; got != want {
		t.Fatalf("quickSightGroupMembershipImportID() = %q, want %q", got, want)
	}
	if got, want := quickSightFolderImportID("123456789012", "folder-1"), "123456789012,folder-1"; got != want {
		t.Fatalf("quickSightFolderImportID() = %q, want %q", got, want)
	}
	if got, want := quickSightFolderMembershipImportID("123456789012", "folder-1", "DASHBOARD", "dash-1"), "123456789012,folder-1,DASHBOARD,dash-1"; got != want {
		t.Fatalf("quickSightFolderMembershipImportID() = %q, want %q", got, want)
	}
	if got, want := quickSightVPCConnectionImportID("123456789012", "vpc-conn"), "123456789012,vpc-conn"; got != want {
		t.Fatalf("quickSightVPCConnectionImportID() = %q, want %q", got, want)
	}
}

func TestQuickSightPagination(t *testing.T) {
	client := &fakeQuickSightListNamespacesClient{
		pages: []*quicksight.ListNamespacesOutput{
			{
				Namespaces: []quicksighttypes.NamespaceInfoV2{{Name: aws.String("default")}},
				NextToken:  aws.String("next"),
			},
			{
				Namespaces: []quicksighttypes.NamespaceInfoV2{{Name: aws.String("sandbox")}},
			},
		},
	}
	namespaces, err := listQuickSightNamespaces(client, "123456789012")
	if err != nil {
		t.Fatalf("listQuickSightNamespaces() error = %v", err)
	}
	if len(namespaces) != 2 {
		t.Fatalf("listQuickSightNamespaces() len = %d, want 2", len(namespaces))
	}
	if client.calls != 2 {
		t.Fatalf("ListNamespaces calls = %d, want 2", client.calls)
	}
}

func TestNewQuickSightResources(t *testing.T) {
	namespace, ok := newQuickSightNamespaceResource("123456789012", quicksighttypes.NamespaceInfoV2{
		CreationStatus: quicksighttypes.NamespaceStatusCreated,
		IdentityStore:  quicksighttypes.IdentityStoreQuicksight,
		Name:           aws.String("default"),
	})
	assertQuickSightResource(t, namespace, ok, "123456789012,default", quickSightNamespaceResourceType)

	group, ok := newQuickSightGroupResource("123456789012", "default", quicksighttypes.Group{
		Description: aws.String("BI authors"),
		GroupName:   aws.String("authors"),
	})
	assertQuickSightResource(t, group, ok, "123456789012/default/authors", quickSightGroupResourceType)

	membership, ok := newQuickSightGroupMembershipResource("123456789012", "default", "authors", quicksighttypes.GroupMember{
		MemberName: aws.String("alice"),
	})
	assertQuickSightResource(t, membership, ok, "123456789012/default/authors/alice", quickSightGroupMembershipResourceType)

	folder, ok := newQuickSightFolderResource("123456789012", quicksighttypes.FolderSummary{
		FolderId:   aws.String("folder-1"),
		FolderType: quicksighttypes.FolderTypeShared,
		Name:       aws.String("Executive"),
	})
	assertQuickSightResource(t, folder, ok, "123456789012,folder-1", quickSightFolderResourceType)

	folderMembership, ok := newQuickSightFolderMembershipResource("123456789012", "folder-1", quicksighttypes.MemberIdArnPair{
		MemberArn: aws.String("arn:aws:quicksight:us-east-1:123456789012:dashboard/dash-1"),
		MemberId:  aws.String("dash-1"),
	})
	assertQuickSightResource(t, folderMembership, ok, "123456789012,folder-1,DASHBOARD,dash-1", quickSightFolderMembershipResourceType)

	vpcConnection, ok := newQuickSightVPCConnectionResource("123456789012", quicksighttypes.VPCConnectionSummary{
		AvailabilityStatus: quicksighttypes.VPCConnectionAvailabilityStatusAvailable,
		Name:               aws.String("analytics"),
		RoleArn:            aws.String("arn:aws:iam::123456789012:role/quicksight-vpc"),
		Status:             quicksighttypes.VPCConnectionResourceStatusCreationSuccessful,
		VPCConnectionId:    aws.String("vpc-conn"),
	})
	assertQuickSightResource(t, vpcConnection, ok, "123456789012,vpc-conn", quickSightVPCConnectionResourceType)

	if _, ok := newQuickSightNamespaceResource("123456789012", quicksighttypes.NamespaceInfoV2{
		CreationStatus: quicksighttypes.NamespaceStatusCreating,
		Name:           aws.String("default"),
	}); ok {
		t.Fatal("creating namespace should be skipped")
	}
	unavailableVPCConnection, ok := newQuickSightVPCConnectionResource("123456789012", quicksighttypes.VPCConnectionSummary{
		AvailabilityStatus: quicksighttypes.VPCConnectionAvailabilityStatusUnavailable,
		Name:               aws.String("analytics"),
		RoleArn:            aws.String("arn:aws:iam::123456789012:role/quicksight-vpc"),
		Status:             quicksighttypes.VPCConnectionResourceStatusCreationSuccessful,
		VPCConnectionId:    aws.String("vpc-conn"),
	})
	assertQuickSightResource(t, unavailableVPCConnection, ok, "123456789012,vpc-conn", quickSightVPCConnectionResourceType)
	if _, ok := newQuickSightFolderMembershipResource("123456789012", "folder-1", quicksighttypes.MemberIdArnPair{
		MemberArn: aws.String("arn:aws:quicksight:us-east-1:123456789012:user/default/alice"),
		MemberId:  aws.String("alice"),
	}); ok {
		t.Fatal("unsupported folder member type should be skipped")
	}
}

func TestQuickSightImportableStatuses(t *testing.T) {
	if !quickSightNamespaceImportable(quicksighttypes.NamespaceStatusCreated) || quickSightNamespaceImportable(quicksighttypes.NamespaceStatusCreating) {
		t.Fatal("namespace importability should allow Created only")
	}
	if !quickSightVPCConnectionImportable(quicksighttypes.VPCConnectionResourceStatusUpdateSuccessful) {
		t.Fatal("VPC connection update success should be importable regardless of availability")
	}
	if !quickSightVPCConnectionImportable(quicksighttypes.VPCConnectionResourceStatusCreationSuccessful) {
		t.Fatal("VPC connection creation success should be importable regardless of availability")
	}
	if quickSightVPCConnectionImportable(quicksighttypes.VPCConnectionResourceStatusUpdateInProgress) {
		t.Fatal("updating VPC connection should be skipped")
	}
}

func TestQuickSightFolderMemberTypeFromARN(t *testing.T) {
	cases := map[string]string{
		"arn:aws:quicksight:us-east-1:123456789012:analysis/analysis-1": "ANALYSIS",
		"arn:aws:quicksight:us-east-1:123456789012:dashboard/dash-1":    "DASHBOARD",
		"arn:aws:quicksight:us-east-1:123456789012:dataset/data-set-1":  "DATASET",
		"arn:aws:quicksight:us-east-1:123456789012:datasource/source-1": "DATASOURCE",
		"arn:aws:quicksight:us-east-1:123456789012:topic/topic-1":       "TOPIC",
		"arn:aws:quicksight:us-east-1:123456789012:user/default/alice":  "",
	}
	for arn, want := range cases {
		if got := quickSightFolderMemberTypeFromARN(arn); got != want {
			t.Fatalf("quickSightFolderMemberTypeFromARN(%q) = %q, want %q", arn, got, want)
		}
	}
}

func TestQuickSightShouldLoadResourceHonorsTypedFilters(t *testing.T) {
	g := QuickSightGenerator{}
	for _, serviceName := range quickSightResourceTypes {
		if !g.shouldLoadQuickSightResource(serviceName) {
			t.Fatalf("without typed filters, %s should be loaded", serviceName)
		}
	}

	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "quicksight_group",
		FieldPath:        "id",
		AcceptableValues: []string{"123456789012/default/authors"},
	}}
	for _, serviceName := range quickSightResourceTypes {
		got := g.shouldLoadQuickSightResource(serviceName)
		want := serviceName == "quicksight_group"
		if got != want {
			t.Fatalf("shouldLoadQuickSightResource(%q) = %t, want %t", serviceName, got, want)
		}
	}
}

func TestQuickSightInitialCleanupHonorsTypedFilters(t *testing.T) {
	namespace, ok := newQuickSightNamespaceResource("123456789012", quicksighttypes.NamespaceInfoV2{
		CreationStatus: quicksighttypes.NamespaceStatusCreated,
		Name:           aws.String("default"),
	})
	assertQuickSightResource(t, namespace, ok, "123456789012,default", quickSightNamespaceResourceType)
	group, ok := newQuickSightGroupResource("123456789012", "default", quicksighttypes.Group{GroupName: aws.String("authors")})
	assertQuickSightResource(t, group, ok, "123456789012/default/authors", quickSightGroupResourceType)
	readers, ok := newQuickSightGroupResource("123456789012", "default", quicksighttypes.Group{GroupName: aws.String("readers")})
	assertQuickSightResource(t, readers, ok, "123456789012/default/readers", quickSightGroupResourceType)

	g := QuickSightGenerator{}
	g.Resources = []terraformutils.Resource{namespace, group, readers}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "quicksight_group",
		FieldPath:        "id",
		AcceptableValues: []string{"123456789012/default/authors"},
	}}
	g.InitialCleanup()

	if len(g.Resources) != 1 {
		t.Fatalf("InitialCleanup() resources len = %d, want 1", len(g.Resources))
	}
	if got := g.Resources[0].InstanceInfo.Type; got != quickSightGroupResourceType {
		t.Fatalf("InitialCleanup() kept resource type = %q, want %s", got, quickSightGroupResourceType)
	}
	if got := g.Resources[0].InstanceState.Attributes["group_name"]; got != "authors" {
		t.Fatalf("InitialCleanup() kept group_name = %q, want authors", got)
	}
}

func TestQuickSightInitialCleanupPreservesPostRefreshFilters(t *testing.T) {
	namespace, ok := newQuickSightNamespaceResource("123456789012", quicksighttypes.NamespaceInfoV2{
		CreationStatus: quicksighttypes.NamespaceStatusCreated,
		Name:           aws.String("default"),
	})
	assertQuickSightResource(t, namespace, ok, "123456789012,default", quickSightNamespaceResourceType)
	group, ok := newQuickSightGroupResource("123456789012", "default", quicksighttypes.Group{GroupName: aws.String("authors")})
	assertQuickSightResource(t, group, ok, "123456789012/default/authors", quickSightGroupResourceType)
	readers, ok := newQuickSightGroupResource("123456789012", "default", quicksighttypes.Group{GroupName: aws.String("readers")})
	assertQuickSightResource(t, readers, ok, "123456789012/default/readers", quickSightGroupResourceType)

	g := QuickSightGenerator{}
	g.Resources = []terraformutils.Resource{namespace, group, readers}
	g.Filter = []terraformutils.ResourceFilter{{
		ServiceName:      "quicksight_group",
		FieldPath:        "arn",
		AcceptableValues: []string{"arn:aws:quicksight:us-east-1:123456789012:group/default/authors"},
	}}
	g.InitialCleanup()

	if len(g.Resources) != 2 {
		t.Fatalf("typed post-refresh InitialCleanup() resources len = %d, want 2", len(g.Resources))
	}
	for _, resource := range g.Resources {
		if got := resource.InstanceInfo.Type; got != quickSightGroupResourceType {
			t.Fatalf("typed post-refresh InitialCleanup() kept resource type = %q, want %s", got, quickSightGroupResourceType)
		}
	}

	g.Resources = []terraformutils.Resource{namespace, group, readers}
	g.Filter = []terraformutils.ResourceFilter{{
		FieldPath:        "tags.env",
		AcceptableValues: []string{"prod"},
	}}
	g.InitialCleanup()

	if len(g.Resources) != 3 {
		t.Fatalf("global post-refresh InitialCleanup() resources len = %d, want 3", len(g.Resources))
	}
}

func TestQuickSightResourceNameUniqueness(t *testing.T) {
	first := terraformutils.TfSanitize(quickSightResourceName("ab", "c"))
	second := terraformutils.TfSanitize(quickSightResourceName("a", "bc"))
	if first == second {
		t.Fatalf("quickSightResourceName() collision after sanitize: %q", first)
	}
}

type fakeQuickSightListNamespacesClient struct {
	pages []*quicksight.ListNamespacesOutput
	calls int
}

func (f *fakeQuickSightListNamespacesClient) ListNamespaces(context.Context, *quicksight.ListNamespacesInput, ...func(*quicksight.Options)) (*quicksight.ListNamespacesOutput, error) {
	if f.calls >= len(f.pages) {
		return &quicksight.ListNamespacesOutput{}, nil
	}
	page := f.pages[f.calls]
	f.calls++
	return page, nil
}

func assertQuickSightResource(t *testing.T, resource terraformutils.Resource, ok bool, wantID, wantType string) {
	t.Helper()
	if !ok {
		t.Fatal("resource should be created")
	}
	if got := resource.InstanceState.ID; got != wantID {
		t.Fatalf("resource ID = %q, want %q", got, wantID)
	}
	if got := resource.InstanceInfo.Type; got != wantType {
		t.Fatalf("resource type = %q, want %q", got, wantType)
	}
	if resource.ResourceName == "" {
		t.Fatal("resource name should not be empty")
	}
}
