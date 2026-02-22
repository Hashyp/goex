package main

import (
	"context"
	"fmt"
	"log"
	"sort"

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

	for _, container := range baseContainers {
		data = append(data, seedBlob{
			container: container,
			name:      "root-file.txt",
			content:   fmt.Sprintf("container=%s root\n", container),
		})
		data = append(data, seedBlob{
			container: container,
			name:      ".hidden-root.txt",
			content:   fmt.Sprintf("hidden root in %s\n", container),
		})
		data = append(data, seedBlob{
			container: container,
			name:      "configs/.secrets/app.env",
			content:   fmt.Sprintf("CONTAINER=%s\n", container),
		})

		for rootFileIndex := 1; rootFileIndex <= bulkRootFiles; rootFileIndex++ {
			data = append(data, seedBlob{
				container: container,
				name:      fmt.Sprintf("root-%03d.txt", rootFileIndex),
				content:   fmt.Sprintf("root file %03d for %s\n", rootFileIndex, container),
			})
		}

		for folderIndex := 1; folderIndex <= bulkFoldersPerContainer; folderIndex++ {
			folderName := fmt.Sprintf("folder-%03d", folderIndex)

			for fileIndex := 1; fileIndex <= bulkFilesPerFolder; fileIndex++ {
				data = append(data, seedBlob{
					container: container,
					name:      fmt.Sprintf("%s/file-%03d.txt", folderName, fileIndex),
					content:   fmt.Sprintf("container=%s folder=%03d file=%03d\n", container, folderIndex, fileIndex),
				})
			}

			// Hidden segment for toggling hidden-entry behavior in Azure pane.
			data = append(data, seedBlob{
				container: container,
				name:      fmt.Sprintf("%s/.meta/hidden-%03d.json", folderName, folderIndex),
				content:   fmt.Sprintf("{\"container\":\"%s\",\"folder\":%d}\n", container, folderIndex),
			})
		}
	}

	// Keep a few semantic samples used in manual UX checks.
	data = append(data,
		seedBlob{container: "goex-dev", name: "docs/readme.md", content: "# docs\n"},
		seedBlob{container: "goex-dev", name: "docs/specs/v1.txt", content: "spec v1\n"},
		seedBlob{container: "media", name: "images/logo.png", content: "fakepng\n"},
		seedBlob{container: "media", name: "images/icons/app.svg", content: "<svg></svg>\n"},
		seedBlob{container: "media", name: "videos/demo.txt", content: "demo\n"},
	)

	return data
}
