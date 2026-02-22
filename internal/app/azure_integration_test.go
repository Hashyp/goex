package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	tea "github.com/charmbracelet/bubbletea"

	"defaultdevcontainer/internal/azureblob"
)

func TestAzureBackendListsContainersAndVirtualFolders(t *testing.T) {
	if os.Getenv("GOEX_RUN_AZURITE_TESTS") != "1" {
		t.Skip("set GOEX_RUN_AZURITE_TESTS=1 to run Azurite integration tests")
	}

	ctx := context.Background()
	client, err := azureblob.NewClient()
	if err != nil {
		t.Fatalf("create azurite client: %v", err)
	}

	containerName := fmt.Sprintf("goexit%d", time.Now().UnixNano())
	if len(containerName) > 63 {
		containerName = containerName[:63]
	}

	if err := azureblob.EnsureContainer(ctx, client, containerName); err != nil {
		t.Fatalf("ensure test container: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteContainer(context.Background(), containerName, nil)
	})

	uploadBlob(t, ctx, client, containerName, "root.txt", "root")
	uploadBlob(t, ctx, client, containerName, "docs/readme.md", "docs")
	uploadBlob(t, ctx, client, containerName, "docs/specs/v1.txt", "spec")
	uploadBlob(t, ctx, client, containerName, "configs/.secrets/app.env", "hidden-segment")
	uploadBlob(t, ctx, client, containerName, ".hidden-root.txt", "hidden-root")

	backend := NewAzureBlobBackend(client)

	containers, err := backend.List(ctx, AzureLocation{Mode: AzureModeContainers}, true)
	if err != nil {
		t.Fatalf("list containers: %v", err)
	}

	var foundContainer bool
	for _, entry := range containers {
		if entry.Name == containerName && entry.Kind == KindContainer {
			foundContainer = true
			break
		}
	}
	if !foundContainer {
		t.Fatalf("expected to find seeded container %q", containerName)
	}

	rootLocation := AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}
	rootEntries, err := backend.List(ctx, rootLocation, false)
	if err != nil {
		t.Fatalf("list root entries: %v", err)
	}
	rootNames := entryNames(rootEntries)
	if !contains(rootNames, "docs") || !contains(rootNames, "root.txt") {
		t.Fatalf("unexpected root entries: %v", rootNames)
	}
	for _, name := range rootNames {
		if strings.HasPrefix(name, ".") {
			t.Fatalf("did not expect hidden root entry when showHidden=false: %v", rootNames)
		}
	}

	docsLocation := AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: "docs/"}
	docsEntries, err := backend.List(ctx, docsLocation, true)
	if err != nil {
		t.Fatalf("list docs entries: %v", err)
	}
	docsNames := entryNames(docsEntries)
	if !contains(docsNames, "readme.md") || !contains(docsNames, "specs") {
		t.Fatalf("unexpected docs entries: %v", docsNames)
	}

	rootEntriesNoHidden, err := backend.List(ctx, rootLocation, true)
	if err != nil {
		t.Fatalf("list root with hidden: %v", err)
	}
	if !contains(entryNames(rootEntriesNoHidden), ".hidden-root.txt") {
		t.Fatalf("expected hidden root blob when showHidden=true")
	}

	if err := backend.Delete(ctx, rootLocation, Entry{Name: "root.txt", FullPath: "root.txt", Kind: KindObject}); err != nil {
		t.Fatalf("delete root blob: %v", err)
	}
	rootEntriesAfterDelete, err := backend.List(ctx, rootLocation, true)
	if err != nil {
		t.Fatalf("list root after delete: %v", err)
	}
	if contains(entryNames(rootEntriesAfterDelete), "root.txt") {
		t.Fatalf("expected root.txt to be deleted, entries=%v", entryNames(rootEntriesAfterDelete))
	}
}

func TestAzureModelDeleteSelectedFiles(t *testing.T) {
	if os.Getenv("GOEX_RUN_AZURITE_TESTS") != "1" {
		t.Skip("set GOEX_RUN_AZURITE_TESTS=1 to run Azurite integration tests")
	}

	ctx := context.Background()
	client, err := azureblob.NewClient()
	if err != nil {
		t.Fatalf("create azurite client: %v", err)
	}

	containerName := fmt.Sprintf("goexit%d", time.Now().UnixNano())
	if len(containerName) > 63 {
		containerName = containerName[:63]
	}

	if err := azureblob.EnsureContainer(ctx, client, containerName); err != nil {
		t.Fatalf("ensure test container: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteContainer(context.Background(), containerName, nil)
	})

	uploadBlob(t, ctx, client, containerName, "a.txt", "a")
	uploadBlob(t, ctx, client, containerName, "b.txt", "b")
	uploadBlob(t, ctx, client, containerName, "c.txt", "c")

	backend := NewAzureBlobBackend(client)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "c.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after selected delete: %v", got)
	}
}

func TestAzureModelDeleteSingleHighlightedFile(t *testing.T) {
	if os.Getenv("GOEX_RUN_AZURITE_TESTS") != "1" {
		t.Skip("set GOEX_RUN_AZURITE_TESTS=1 to run Azurite integration tests")
	}

	ctx := context.Background()
	client, err := azureblob.NewClient()
	if err != nil {
		t.Fatalf("create azurite client: %v", err)
	}

	containerName := fmt.Sprintf("goexit%d", time.Now().UnixNano())
	if len(containerName) > 63 {
		containerName = containerName[:63]
	}

	if err := azureblob.EnsureContainer(ctx, client, containerName); err != nil {
		t.Fatalf("ensure test container: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteContainer(context.Background(), containerName, nil)
	})

	uploadBlob(t, ctx, client, containerName, "a.txt", "a")
	uploadBlob(t, ctx, client, containerName, "b.txt", "b")

	backend := NewAzureBlobBackend(client)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "b.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after single delete: %v", got)
	}
}

func TestAzureModelDeleteDirectoryRecursively(t *testing.T) {
	if os.Getenv("GOEX_RUN_AZURITE_TESTS") != "1" {
		t.Skip("set GOEX_RUN_AZURITE_TESTS=1 to run Azurite integration tests")
	}

	ctx := context.Background()
	client, err := azureblob.NewClient()
	if err != nil {
		t.Fatalf("create azurite client: %v", err)
	}

	containerName := fmt.Sprintf("goexit%d", time.Now().UnixNano())
	if len(containerName) > 63 {
		containerName = containerName[:63]
	}

	if err := azureblob.EnsureContainer(ctx, client, containerName); err != nil {
		t.Fatalf("ensure test container: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteContainer(context.Background(), containerName, nil)
	})

	uploadBlob(t, ctx, client, containerName, "docs/readme.md", "docs")
	uploadBlob(t, ctx, client, containerName, "docs/specs/v1.txt", "spec")
	uploadBlob(t, ctx, client, containerName, "root.txt", "root")

	backend := NewAzureBlobBackend(client)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModalVisible {
		t.Fatal("expected delete modal for directory")
	}
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, AzureLocation{Mode: AzureModeObjects, Container: containerName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after directory delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "root.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after directory delete: %v", got)
	}
}

func uploadBlob(t *testing.T, ctx context.Context, client *azblob.Client, containerName, blobName, content string) {
	t.Helper()
	_, err := client.UploadBuffer(ctx, containerName, blobName, []byte(content), nil)
	if err != nil {
		t.Fatalf("upload %s/%s: %v", containerName, blobName, err)
	}
}

func entryNames(entries []Entry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	return names
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
