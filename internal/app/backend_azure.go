package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
)

const azureDelimiter = "/"
const maxAzureEntries = 20000

type AzureBlobBackend struct {
	client *azblob.Client
}

func NewAzureBlobBackend(client *azblob.Client) AzureBlobBackend {
	return AzureBlobBackend{client: client}
}

func (b AzureBlobBackend) InitialLocation() Location {
	return AzureLocation{Mode: AzureModeContainers}
}

func (b AzureBlobBackend) DisplayPath(state Location) string {
	azure, ok := state.(AzureLocation)
	if !ok {
		return "azure:<invalid-location>"
	}

	if azure.Mode == AzureModeContainers {
		return "azure:/"
	}

	prefix := azure.Prefix
	if prefix == "" {
		return fmt.Sprintf("azure:/%s", azure.Container)
	}

	return fmt.Sprintf("azure:/%s/%s", azure.Container, prefix)
}

func (b AzureBlobBackend) ParentHighlightName(state Location) string {
	azure, ok := state.(AzureLocation)
	if !ok || azure.Mode != AzureModeObjects {
		return ""
	}

	return parentHighlightName(azure.Prefix, azure.Container, azureDelimiter)
}

func (b AzureBlobBackend) LoadTimeout() time.Duration {
	return 30 * time.Second
}

func (b AzureBlobBackend) List(ctx context.Context, state Location, showHidden bool) ([]Entry, error) {
	azure, ok := state.(AzureLocation)
	if !ok {
		return nil, ErrInvalidLocation
	}
	if b.client == nil {
		return nil, fmt.Errorf("azure client not configured")
	}

	switch azure.Mode {
	case AzureModeContainers:
		return b.listContainers(ctx, showHidden)
	case AzureModeObjects:
		return b.listObjects(ctx, azure.Container, azure.Prefix, showHidden)
	default:
		return nil, fmt.Errorf("unknown azure mode: %s", azure.Mode)
	}
}

func (b AzureBlobBackend) Enter(_ context.Context, state Location, highlighted Entry) (Location, bool, error) {
	azure, ok := state.(AzureLocation)
	if !ok {
		return state, false, ErrInvalidLocation
	}

	switch azure.Mode {
	case AzureModeContainers:
		if highlighted.Kind != KindContainer {
			return state, false, nil
		}
		return AzureLocation{Mode: AzureModeObjects, Container: highlighted.Name, Prefix: ""}, true, nil
	case AzureModeObjects:
		if highlighted.Kind != KindDirectory {
			return state, false, nil
		}

		nextPrefix := enterPrefix(highlighted.FullPath, azureDelimiter)
		return AzureLocation{Mode: AzureModeObjects, Container: azure.Container, Prefix: nextPrefix}, true, nil
	default:
		return state, false, nil
	}
}

func (b AzureBlobBackend) Delete(ctx context.Context, state Location, highlighted Entry) error {
	azure, ok := state.(AzureLocation)
	if !ok {
		return ErrInvalidLocation
	}
	if b.client == nil {
		return fmt.Errorf("azure client not configured")
	}
	if azure.Mode != AzureModeObjects || highlighted.Kind != KindObject {
		return nil
	}
	if azure.Container == "" {
		return fmt.Errorf("azure container not selected")
	}
	blobName := highlighted.FullPath
	if blobName == "" {
		blobName = highlighted.Name
	}
	if blobName == "" {
		return fmt.Errorf("azure object path is empty")
	}

	_, err := b.client.DeleteBlob(ctx, azure.Container, blobName, nil)
	return err
}

func (b AzureBlobBackend) Parent(state Location) (Location, bool) {
	azure, ok := state.(AzureLocation)
	if !ok {
		return state, false
	}

	if azure.Mode == AzureModeContainers {
		return state, false
	}

	if azure.Prefix == "" {
		return AzureLocation{Mode: AzureModeContainers}, true
	}

	parent := parentPrefix(azure.Prefix, azureDelimiter)
	return AzureLocation{Mode: AzureModeObjects, Container: azure.Container, Prefix: parent}, true
}

func (b AzureBlobBackend) listContainers(ctx context.Context, showHidden bool) ([]Entry, error) {
	serviceClient := b.client.ServiceClient()
	pager := serviceClient.NewListContainersPager(nil)

	entries := []Entry{}
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range resp.ContainerItems {
			if item == nil || item.Name == nil {
				continue
			}
			name := *item.Name
			if !showHidden && isHiddenBySegment(name) {
				continue
			}

			entry := Entry{
				ID:       "container:" + name,
				Name:     name,
				FullPath: name,
				Kind:     KindContainer,
			}
			if item.Properties != nil && item.Properties.LastModified != nil {
				entry.ModTime = *item.Properties.LastModified
				entry.HasModTime = true
			}

			entries = append(entries, entry)
			if len(entries) > maxAzureEntries {
				return nil, fmt.Errorf("azure list exceeded max entries limit (%d)", maxAzureEntries)
			}
		}
	}

	sortEntries(entries)
	return entries, nil
}

func (b AzureBlobBackend) listObjects(ctx context.Context, containerName string, prefix string, showHidden bool) ([]Entry, error) {
	if containerName == "" {
		return nil, fmt.Errorf("azure container not selected")
	}

	containerClient := b.client.ServiceClient().NewContainerClient(containerName)
	options := &container.ListBlobsHierarchyOptions{Prefix: to.Ptr(prefix)}
	pager := containerClient.NewListBlobsHierarchyPager(azureDelimiter, options)

	entries := []Entry{}
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Segment == nil {
			continue
		}

		for _, item := range resp.Segment.BlobPrefixes {
			if item == nil || item.Name == nil {
				continue
			}

			fullPrefix := strings.TrimSuffix(*item.Name, azureDelimiter)
			displayName := trimPrefix(fullPrefix, prefix)
			displayName = strings.TrimSuffix(displayName, azureDelimiter)
			if displayName == "" {
				continue
			}
			if !showHidden && isHiddenBySegment(displayName) {
				continue
			}

			entries = append(entries, Entry{
				ID:       "dir:" + containerName + "/" + fullPrefix,
				Name:     displayName,
				FullPath: fullPrefix,
				Kind:     KindDirectory,
			})
			if len(entries) > maxAzureEntries {
				return nil, fmt.Errorf("azure list exceeded max entries limit (%d)", maxAzureEntries)
			}
		}

		for _, item := range resp.Segment.BlobItems {
			if item == nil || item.Name == nil {
				continue
			}

			fullName := *item.Name
			displayName := trimPrefix(fullName, prefix)
			if displayName == "" || strings.Contains(displayName, azureDelimiter) {
				continue
			}
			if !showHidden && isHiddenBySegment(displayName) {
				continue
			}

			entry := Entry{
				ID:       "blob:" + containerName + "/" + fullName,
				Name:     displayName,
				FullPath: fullName,
				Kind:     KindObject,
			}
			if item.Properties != nil {
				if item.Properties.ContentLength != nil {
					entry.SizeBytes = *item.Properties.ContentLength
				}
				if item.Properties.LastModified != nil {
					entry.ModTime = *item.Properties.LastModified
					entry.HasModTime = true
				}
			}

			entries = append(entries, entry)
			if len(entries) > maxAzureEntries {
				return nil, fmt.Errorf("azure list exceeded max entries limit (%d)", maxAzureEntries)
			}
		}
	}

	sortEntries(entries)
	return entries, nil
}

func isHiddenBySegment(path string) bool {
	return hiddenBySegment(path, azureDelimiter)
}
