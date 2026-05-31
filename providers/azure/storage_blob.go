// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v4"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	blobFormatString = `https://%s.blob.core.windows.net`
	blobIDFormat     = `https://%s.blob.core.windows.net/%s/%s`
)

type StorageBlobGenerator struct {
	AzureService
}

func (g StorageBlobGenerator) getAccountPrimaryKey(ctx context.Context, accountName, accountGroupName string) (string, error) {
	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)

	storageAccountsClient, err := armstorage.NewAccountsClient(subscriptionID, credential, clientOptions)
	if err != nil {
		return "", fmt.Errorf("create storage accounts client: %w", err)
	}

	response, err := storageAccountsClient.ListKeys(ctx, accountGroupName, accountName, nil)
	if err != nil {
		return "", fmt.Errorf("list keys for storage account %q in resource group %q: %w", accountName, accountGroupName, err)
	}
	if len(response.Keys) == 0 || response.Keys[0].Value == nil {
		return "", fmt.Errorf("storage account %q in resource group %q returned no primary key", accountName, accountGroupName)
	}
	return *response.Keys[0].Value, nil
}

func (g StorageBlobGenerator) getContainerURL(ctx context.Context, accountName, accountGroupName, containerName string) (azblob.ContainerURL, error) {
	accountPrimaryKey, err := g.getAccountPrimaryKey(ctx, accountName, accountGroupName)
	if err != nil {
		return azblob.ContainerURL{}, err
	}
	sharedKeyCredential, err := azblob.NewSharedKeyCredential(accountName, accountPrimaryKey)
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	p := azblob.NewPipeline(sharedKeyCredential, azblob.PipelineOptions{})
	accountURL, err := url.Parse(fmt.Sprintf(blobFormatString, accountName))
	if err != nil {
		return azblob.ContainerURL{}, err
	}

	serviceURL := azblob.NewServiceURL(*accountURL, p)
	containerURL := serviceURL.NewContainerURL(containerName)

	return containerURL, nil
}

func (g StorageBlobGenerator) getBlobsFromContainer(ctx context.Context, accountName, accountGroupName, containerName string) ([]azblob.BlobItemInternal, error) {
	containerURL, err := g.getContainerURL(ctx, accountName, accountGroupName, containerName)
	if err != nil {
		return nil, err
	}

	blobListResponse, err := containerURL.ListBlobsFlatSegment(
		ctx,
		azblob.Marker{},
		azblob.ListBlobsSegmentOptions{
			Details: azblob.BlobListingDetails{
				Snapshots: true,
			},
		})
	if err != nil {
		return nil, err
	}

	return blobListResponse.Segment.BlobItems, nil
}

func (g StorageBlobGenerator) listStorageBlobs() ([]terraformutils.Resource, error) {
	var storageBlobsResources []terraformutils.Resource
	ctx := context.Background()

	subscriptionID := g.Args["config"].(providerConfig).SubscriptionID
	credential := g.Args["credential"].(azcore.TokenCredential)
	clientOptions := g.Args["clientOptions"].(*arm.ClientOptions)
	resourceGroup := g.Args["resource_group"].(string)
	blobContainerGenerator := NewStorageContainerGenerator(subscriptionID, credential, clientOptions, resourceGroup)
	blobContainersResources, err := blobContainerGenerator.ListBlobContainers()
	if err != nil {
		return storageBlobsResources, err
	}

	for _, blobContainerResource := range blobContainersResources {
		containerID := blobContainerResource.InstanceState.ID
		parsedContainerID, err := ParseAzureResourceID(containerID)
		if err != nil {
			return storageBlobsResources, err
		}

		storageAccountName := blobContainerResource.InstanceState.Attributes["storage_account_name"]
		containerName := blobContainerResource.InstanceState.Attributes["name"]
		blobsList, err := g.getBlobsFromContainer(ctx, storageAccountName, parsedContainerID.ResourceGroup, containerName)
		if err != nil {
			return storageBlobsResources, err
		}

		for _, blobItem := range blobsList {
			storageBlobsResources = append(storageBlobsResources, terraformutils.NewSimpleResource(
				fmt.Sprintf(blobIDFormat, storageAccountName, containerName, blobItem.Name),
				blobItem.Name,
				"azurerm_storage_blob",
				"azurerm",
				[]string{}))
		}
	}

	return storageBlobsResources, err
}

func (g *StorageBlobGenerator) InitResources() error {
	resources, err := g.listStorageBlobs()
	if err != nil {
		return err
	}

	g.Resources = append(g.Resources, resources...)

	return nil
}
