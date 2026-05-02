// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"google.golang.org/api/iterator"
)

func TestCreateServiceAccountResourcesReturnsIteratorErrors(t *testing.T) {
	iteratorErr := errors.New("page failed")
	generator := IamGenerator{}

	_, err := generator.createServiceAccountResources(&fakeServiceAccountIterator{
		responses: []fakeServiceAccountResponse{
			{err: iteratorErr},
		},
	})
	if err == nil {
		t.Fatal("expected service account iterator error")
	}
	if !strings.Contains(err.Error(), "list iam service accounts") {
		t.Fatalf("error = %q, want service account context", err)
	}
	if !errors.Is(err, iteratorErr) {
		t.Fatalf("error does not wrap iterator error: %v", err)
	}
}

func TestCreateServiceAccountResourcesSkipsInvalidEmails(t *testing.T) {
	generator := IamGenerator{}

	resources, err := generator.createServiceAccountResources(&fakeServiceAccountIterator{
		responses: []fakeServiceAccountResponse{
			{
				serviceAccount: &adminpb.ServiceAccount{
					Email:    "valid@example.iam.gserviceaccount.com",
					Name:     "projects/test/serviceAccounts/valid@example.iam.gserviceaccount.com",
					UniqueId: "123",
				},
			},
			{
				serviceAccount: &adminpb.ServiceAccount{
					Email:    "_invalid@example.iam.gserviceaccount.com",
					Name:     "projects/test/serviceAccounts/_invalid@example.iam.gserviceaccount.com",
					UniqueId: "456",
				},
			},
			{err: iterator.Done},
		},
	})
	if err != nil {
		t.Fatalf("createServiceAccountResources returned error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	resource := resources[0]
	if got := resource.InstanceState.ID; got != "projects/test/serviceAccounts/valid@example.iam.gserviceaccount.com" {
		t.Fatalf("resource ID = %q, want service account name", got)
	}
	if got := resource.ResourceName; got != "tfer--123" {
		t.Fatalf("resource name = %q, want sanitized service account unique ID", got)
	}
}

type fakeServiceAccountResponse struct {
	serviceAccount *adminpb.ServiceAccount
	err            error
}

type fakeServiceAccountIterator struct {
	responses []fakeServiceAccountResponse
	index     int
}

func (i *fakeServiceAccountIterator) Next() (*adminpb.ServiceAccount, error) {
	if i.index >= len(i.responses) {
		return nil, iterator.Done
	}
	response := i.responses[i.index]
	i.index++
	return response.serviceAccount, response.err
}
