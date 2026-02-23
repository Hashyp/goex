package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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
	if azure.Mode != AzureModeObjects || !isDeleteTargetKind(highlighted.Kind) {
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

	if highlighted.Kind == KindDirectory {
		return b.deletePrefixRecursive(ctx, azure.Container, blobName)
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

func (b AzureBlobBackend) deletePrefixRecursive(ctx context.Context, containerName string, prefix string) error {
	queryPrefix := enterPrefix(prefix, azureDelimiter)
	blobNames, err := b.listBlobNamesByPrefix(ctx, containerName, queryPrefix)
	if err != nil {
		return err
	}

	// Marker blobs can exist as both "dir" and "dir/".
	blobNames = append(blobNames, prefix, queryPrefix)
	blobNames = uniqueStrings(blobNames)
	for _, blobName := range blobNames {
		_, err := b.client.DeleteBlob(ctx, containerName, blobName, nil)
		if err != nil {
			if isAzureNotFoundError(err) {
				continue
			}
			return err
		}
	}

	return nil
}

func isAzureNotFoundError(err error) bool {
	var responseErr *azcore.ResponseError
	if !errors.As(err, &responseErr) {
		return false
	}

	return responseErr.StatusCode == 404 || responseErr.ErrorCode == "BlobNotFound"
}

func (b AzureBlobBackend) listBlobNamesByPrefix(ctx context.Context, containerName string, prefix string) ([]string, error) {
	containerClient := b.client.ServiceClient().NewContainerClient(containerName)
	pager := containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: to.Ptr(prefix),
	})

	blobNames := make([]string, 0)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Segment == nil {
			continue
		}
		for _, item := range resp.Segment.BlobItems {
			if item == nil || item.Name == nil {
				continue
			}
			blobNames = append(blobNames, *item.Name)
		}
	}

	return blobNames, nil
}

func (b AzureBlobBackend) EnumerateCopy(ctx context.Context, state Location, selected []Entry, destination Location) ([]TransferPlanItem, error) {
	azure, ok := state.(AzureLocation)
	if !ok {
		return nil, ErrInvalidLocation
	}
	if b.client == nil {
		return nil, fmt.Errorf("azure client not configured")
	}

	plan := make([]TransferPlanItem, 0, len(selected))
	for _, entry := range selected {
		switch azure.Mode {
		case AzureModeContainers:
			if entry.Kind != KindContainer || entry.Name == "" {
				continue
			}
			blobNames, err := b.listBlobNamesByPrefix(ctx, entry.Name, "")
			if err != nil {
				return nil, err
			}
			for _, blobName := range blobNames {
				srcRef := TransferObjectRef{
					Provider: "azure",
					Scope:    entry.Name,
					Path:     blobName,
					Display:  "azure:/" + entry.Name + "/" + blobName,
				}
				dstRef, err := resolveDestinationRef(destination, path.Join(entry.Name, blobName))
				if err != nil {
					return nil, err
				}
				plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
			}
		case AzureModeObjects:
			if azure.Container == "" {
				return nil, fmt.Errorf("azure container not selected")
			}
			switch entry.Kind {
			case KindObject:
				blobName := entry.FullPath
				if blobName == "" {
					blobName = joinObjectPath(azure.Prefix, entry.Name)
				}
				srcRef, err := sourceRefForLocation(azure, blobName)
				if err != nil {
					return nil, err
				}
				dstRef, err := resolveDestinationRef(destination, entry.Name)
				if err != nil {
					return nil, err
				}
				plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
			case KindDirectory:
				prefix := enterPrefix(entry.FullPath, azureDelimiter)
				blobNames, err := b.listBlobNamesByPrefix(ctx, azure.Container, prefix)
				if err != nil {
					return nil, err
				}
				for _, blobName := range blobNames {
					rel := path.Join(entry.Name, trimPrefix(blobName, prefix))
					srcRef, err := sourceRefForLocation(azure, blobName)
					if err != nil {
						return nil, err
					}
					dstRef, err := resolveDestinationRef(destination, rel)
					if err != nil {
						return nil, err
					}
					plan = append(plan, TransferPlanItem{Source: srcRef, Destination: dstRef})
				}
			}
		default:
			return nil, fmt.Errorf("unknown azure mode: %s", azure.Mode)
		}
	}

	return plan, nil
}

func (b AzureBlobBackend) OpenCopyReader(ctx context.Context, source TransferObjectRef) (CopyReadHandle, error) {
	if b.client == nil {
		return CopyReadHandle{}, fmt.Errorf("azure client not configured")
	}
	resp, err := b.client.DownloadStream(ctx, source.Scope, source.Path, nil)
	if err != nil {
		return CopyReadHandle{}, err
	}

	reader := resp.NewRetryReader(ctx, nil)
	handle := CopyReadHandle{Reader: reader}
	if resp.ContentLength != nil {
		handle.Metadata.SizeBytes = *resp.ContentLength
	}
	if resp.LastModified != nil {
		handle.Metadata.ModTime = *resp.LastModified
		handle.Metadata.HasModTime = true
	}
	return handle, nil
}

func (b AzureBlobBackend) CopyDestinationExists(ctx context.Context, destination TransferObjectRef) (bool, error) {
	if b.client == nil {
		return false, fmt.Errorf("azure client not configured")
	}
	containerClient := b.client.ServiceClient().NewContainerClient(destination.Scope)
	_, err := containerClient.NewBlobClient(destination.Path).GetProperties(ctx, nil)
	if err == nil {
		return true, nil
	}
	if isAzureNotFoundError(err) {
		return false, nil
	}

	return false, err
}

func (b AzureBlobBackend) OpenCopyWriter(ctx context.Context, destination TransferObjectRef, _ TransferObjectMetadata) (io.WriteCloser, error) {
	if b.client == nil {
		return nil, fmt.Errorf("azure client not configured")
	}

	reader, writer := io.Pipe()
	done := make(chan error, 1)
	go func() {
		_, err := b.client.UploadStream(ctx, destination.Scope, destination.Path, reader, nil)
		done <- err
	}()

	return &pipeUploadWriteCloser{pipe: writer, done: done}, nil
}
