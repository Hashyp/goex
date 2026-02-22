package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"defaultdevcontainer/internal/azureblob"
)

type seedBlob struct {
	container string
	name      string
	content   string
}

const (
	bulkFoldersPerContainer = 30
	bulkFilesPerFolder      = 22
	bulkRootFiles           = 12
	progressLogEvery        = 250
	containerPrefix         = "az-"
	objectPrefix            = "az_"
)

var baseContainers = []string{
	"goex-dev",
	"media",
	"finance",
	"logs",
	"reports",
	"archive",
	"datasets",
}

func main() {
	ctx := context.Background()

	client, err := azureblob.NewClient()
	if err != nil {
		log.Fatalf("create azurite client: %v", err)
	}

	seedData := generateSeedData()
	if err := seed(ctx, client, seedData); err != nil {
		log.Fatal(err)
	}

	log.Println("Azurite seed complete")
}

func seed(ctx context.Context, client *azblob.Client, data []seedBlob) error {
	containerSet := map[string]struct{}{}
	for _, item := range data {
		containerSet[item.container] = struct{}{}
	}

	containers := make([]string, 0, len(containerSet))
	for name := range containerSet {
		containers = append(containers, name)
	}
	sort.Strings(containers)

	for _, containerName := range containers {
		if err := azureblob.EnsureContainer(ctx, client, containerName); err != nil {
			return fmt.Errorf("ensure container %q: %w", containerName, err)
		}
	}

	log.Printf("Seeding %d blobs across %d containers", len(data), len(containers))

	for idx, item := range data {
		_, err := client.UploadBuffer(ctx, item.container, item.name, []byte(item.content), nil)
		if err != nil {
			return fmt.Errorf("upload %s/%s: %w", item.container, item.name, err)
		}
		if (idx+1)%progressLogEvery == 0 || idx+1 == len(data) {
			log.Printf("Seed progress: %d/%d", idx+1, len(data))
		}
	}

	return nil
}

func generateSeedData() []seedBlob {
	var data []seedBlob

	for _, baseContainer := range baseContainers {
		container := prefixedContainer(baseContainer)
		data = append(data, seedBlob{
			container: container,
			name:      prefixedPath("root-file.txt"),
			content:   fmt.Sprintf("container=%s root\n", container),
		})
		data = append(data, seedBlob{
			container: container,
			name:      prefixedPath(".hidden-root.txt"),
			content:   fmt.Sprintf("hidden root in %s\n", container),
		})
		data = append(data, seedBlob{
			container: container,
			name:      prefixedPath("configs/.secrets/app.env"),
			content:   fmt.Sprintf("CONTAINER=%s\n", container),
		})

		for rootFileIndex := 1; rootFileIndex <= bulkRootFiles; rootFileIndex++ {
			data = append(data, seedBlob{
				container: container,
				name:      prefixedPath(fmt.Sprintf("root-%03d.txt", rootFileIndex)),
				content:   fmt.Sprintf("root file %03d for %s\n", rootFileIndex, container),
			})
		}

		for folderIndex := 1; folderIndex <= bulkFoldersPerContainer; folderIndex++ {
			for fileIndex := 1; fileIndex <= bulkFilesPerFolder; fileIndex++ {
				data = append(data, seedBlob{
					container: container,
					name: prefixedPath(
						fmt.Sprintf("folder-%03d/file-%03d.txt", folderIndex, fileIndex),
					),
					content: fmt.Sprintf("container=%s folder=%03d file=%03d\n", container, folderIndex, fileIndex),
				})
			}

			// Hidden segment for toggling hidden-entry behavior in Azure pane.
			data = append(data, seedBlob{
				container: container,
				name: prefixedPath(
					fmt.Sprintf("folder-%03d/.meta/hidden-%03d.json", folderIndex, folderIndex),
				),
				content: fmt.Sprintf("{\"container\":\"%s\",\"folder\":%d}\n", container, folderIndex),
			})
		}
	}

	// Keep a few semantic samples used in manual UX checks.
	data = append(data,
		seedBlob{container: prefixedContainer("goex-dev"), name: prefixedPath("docs/readme.md"), content: "# docs\n"},
		seedBlob{container: prefixedContainer("goex-dev"), name: prefixedPath("docs/specs/v1.txt"), content: "spec v1\n"},
		seedBlob{container: prefixedContainer("media"), name: prefixedPath("images/logo.png"), content: "fakepng\n"},
		seedBlob{container: prefixedContainer("media"), name: prefixedPath("images/icons/app.svg"), content: "<svg></svg>\n"},
		seedBlob{container: prefixedContainer("media"), name: prefixedPath("videos/demo.txt"), content: "demo\n"},
	)

	return data
}

func prefixedContainer(name string) string {
	return containerPrefix + name
}

func prefixedPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = prefixedSegment(part)
	}
	return strings.Join(parts, "/")
}

func prefixedSegment(name string) string {
	if strings.HasPrefix(name, ".") {
		return "." + objectPrefix + strings.TrimPrefix(name, ".")
	}
	return objectPrefix + name
}
