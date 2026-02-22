package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"defaultdevcontainer/internal/s3blob"
)

type seedObject struct {
	bucket  string
	key     string
	content string
}

const (
	bulkFoldersPerBucket = 30
	bulkFilesPerFolder   = 22
	bulkRootFiles        = 12
	progressLogEvery     = 250
)

var baseBuckets = []string{
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

	cfg := s3blob.DefaultConfig()
	client, err := s3blob.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("create s3 client: %v", err)
	}

	seedData := generateSeedData()
	if err := seed(ctx, client, seedData); err != nil {
		log.Fatal(err)
	}

	log.Println("MinIO seed complete")
}

func seed(ctx context.Context, client *s3.Client, data []seedObject) error {
	bucketSet := map[string]struct{}{}
	for _, item := range data {
		bucketSet[item.bucket] = struct{}{}
	}

	buckets := make([]string, 0, len(bucketSet))
	for name := range bucketSet {
		buckets = append(buckets, name)
	}
	sort.Strings(buckets)

	for _, bucketName := range buckets {
		if err := s3blob.EnsureBucket(ctx, client, bucketName); err != nil {
			return fmt.Errorf("ensure bucket %q: %w", bucketName, err)
		}
	}

	log.Printf("Seeding %d objects across %d buckets", len(data), len(buckets))
	for idx, item := range data {
		_, err := client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(item.bucket),
			Key:    aws.String(item.key),
			Body:   strings.NewReader(item.content),
		})
		if err != nil {
			return fmt.Errorf("put %s/%s: %w", item.bucket, item.key, err)
		}
		if (idx+1)%progressLogEvery == 0 || idx+1 == len(data) {
			log.Printf("Seed progress: %d/%d", idx+1, len(data))
		}
	}

	return nil
}

func generateSeedData() []seedObject {
	var data []seedObject

	for _, bucket := range baseBuckets {
		data = append(data, seedObject{
			bucket:  bucket,
			key:     "root-file.txt",
			content: fmt.Sprintf("bucket=%s root\n", bucket),
		})
		data = append(data, seedObject{
			bucket:  bucket,
			key:     ".hidden-root.txt",
			content: fmt.Sprintf("hidden root in %s\n", bucket),
		})
		data = append(data, seedObject{
			bucket:  bucket,
			key:     "configs/.secrets/app.env",
			content: fmt.Sprintf("BUCKET=%s\n", bucket),
		})

		for rootFileIndex := 1; rootFileIndex <= bulkRootFiles; rootFileIndex++ {
			data = append(data, seedObject{
				bucket:  bucket,
				key:     fmt.Sprintf("root-%03d.txt", rootFileIndex),
				content: fmt.Sprintf("root file %03d for %s\n", rootFileIndex, bucket),
			})
		}

		for folderIndex := 1; folderIndex <= bulkFoldersPerBucket; folderIndex++ {
			folderName := fmt.Sprintf("folder-%03d", folderIndex)

			for fileIndex := 1; fileIndex <= bulkFilesPerFolder; fileIndex++ {
				data = append(data, seedObject{
					bucket:  bucket,
					key:     fmt.Sprintf("%s/file-%03d.txt", folderName, fileIndex),
					content: fmt.Sprintf("bucket=%s folder=%03d file=%03d\n", bucket, folderIndex, fileIndex),
				})
			}

			data = append(data, seedObject{
				bucket:  bucket,
				key:     fmt.Sprintf("%s/.meta/hidden-%03d.json", folderName, folderIndex),
				content: fmt.Sprintf("{\"bucket\":\"%s\",\"folder\":%d}\n", bucket, folderIndex),
			})
		}
	}

	data = append(data,
		seedObject{bucket: "goex-dev", key: "docs/readme.md", content: "# docs\n"},
		seedObject{bucket: "goex-dev", key: "docs/specs/v1.txt", content: "spec v1\n"},
		seedObject{bucket: "media", key: "images/logo.png", content: "fakepng\n"},
		seedObject{bucket: "media", key: "images/icons/app.svg", content: "<svg></svg>\n"},
		seedObject{bucket: "media", key: "videos/demo.txt", content: "demo\n"},
	)

	return data
}
