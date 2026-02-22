package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	tea "github.com/charmbracelet/bubbletea"

	"defaultdevcontainer/internal/s3blob"
)

func TestS3BackendListsBucketsAndVirtualFolders(t *testing.T) {
	if os.Getenv("GOEX_RUN_MINIO_TESTS") != "1" {
		t.Skip("set GOEX_RUN_MINIO_TESTS=1 to run MinIO integration tests")
	}

	ctx := context.Background()
	cfg := s3blob.DefaultConfig()
	client, err := s3blob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create minio s3 client: %v", err)
	}

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := s3blob.EnsureBucket(ctx, client, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupS3Bucket(context.Background(), client, bucketName)
	})

	putS3Object(t, ctx, client, bucketName, "root.txt", "root")
	putS3Object(t, ctx, client, bucketName, "docs/readme.md", "docs")
	putS3Object(t, ctx, client, bucketName, "docs/specs/v1.txt", "spec")
	putS3Object(t, ctx, client, bucketName, "configs/.secrets/app.env", "hidden-segment")
	putS3Object(t, ctx, client, bucketName, ".hidden-root.txt", "hidden-root")

	backend := NewS3Backend(client, cfg.RequestTimeout)

	buckets, err := backend.List(ctx, S3Location{Mode: S3ModeBuckets}, true)
	if err != nil {
		t.Fatalf("list buckets: %v", err)
	}

	var foundBucket bool
	for _, entry := range buckets {
		if entry.Name == bucketName && entry.Kind == KindBucket {
			foundBucket = true
			break
		}
	}
	if !foundBucket {
		t.Fatalf("expected to find seeded bucket %q", bucketName)
	}

	rootLocation := S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}
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

	docsLocation := S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: "docs/"}
	docsEntries, err := backend.List(ctx, docsLocation, true)
	if err != nil {
		t.Fatalf("list docs entries: %v", err)
	}
	docsNames := entryNames(docsEntries)
	if !contains(docsNames, "readme.md") || !contains(docsNames, "specs") {
		t.Fatalf("unexpected docs entries: %v", docsNames)
	}

	rootEntriesWithHidden, err := backend.List(ctx, rootLocation, true)
	if err != nil {
		t.Fatalf("list root with hidden: %v", err)
	}
	if !contains(entryNames(rootEntriesWithHidden), ".hidden-root.txt") {
		t.Fatalf("expected hidden root object when showHidden=true")
	}

	if err := backend.Delete(ctx, rootLocation, Entry{Name: "root.txt", FullPath: "root.txt", Kind: KindObject}); err != nil {
		t.Fatalf("delete root object: %v", err)
	}
	rootEntriesAfterDelete, err := backend.List(ctx, rootLocation, true)
	if err != nil {
		t.Fatalf("list root after delete: %v", err)
	}
	if contains(entryNames(rootEntriesAfterDelete), "root.txt") {
		t.Fatalf("expected root.txt to be deleted, entries=%v", entryNames(rootEntriesAfterDelete))
	}
}

func TestS3ModelDeleteSelectedFiles(t *testing.T) {
	if os.Getenv("GOEX_RUN_MINIO_TESTS") != "1" {
		t.Skip("set GOEX_RUN_MINIO_TESTS=1 to run MinIO integration tests")
	}

	ctx := context.Background()
	cfg := s3blob.DefaultConfig()
	client, err := s3blob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create minio s3 client: %v", err)
	}

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := s3blob.EnsureBucket(ctx, client, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupS3Bucket(context.Background(), client, bucketName)
	})

	putS3Object(t, ctx, client, bucketName, "a.txt", "a")
	putS3Object(t, ctx, client, bucketName, "b.txt", "b")
	putS3Object(t, ctx, client, bucketName, "c.txt", "c")

	backend := NewS3Backend(client, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "c.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after selected delete: %v", got)
	}
}

func TestS3ModelDeleteSingleHighlightedFile(t *testing.T) {
	if os.Getenv("GOEX_RUN_MINIO_TESTS") != "1" {
		t.Skip("set GOEX_RUN_MINIO_TESTS=1 to run MinIO integration tests")
	}

	ctx := context.Background()
	cfg := s3blob.DefaultConfig()
	client, err := s3blob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create minio s3 client: %v", err)
	}

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := s3blob.EnsureBucket(ctx, client, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupS3Bucket(context.Background(), client, bucketName)
	})

	putS3Object(t, ctx, client, bucketName, "a.txt", "a")
	putS3Object(t, ctx, client, bucketName, "b.txt", "b")

	backend := NewS3Backend(client, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "b.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after single delete: %v", got)
	}
}

func TestS3ModelDeleteDirectoryRecursively(t *testing.T) {
	if os.Getenv("GOEX_RUN_MINIO_TESTS") != "1" {
		t.Skip("set GOEX_RUN_MINIO_TESTS=1 to run MinIO integration tests")
	}

	ctx := context.Background()
	cfg := s3blob.DefaultConfig()
	client, err := s3blob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create minio s3 client: %v", err)
	}

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := s3blob.EnsureBucket(ctx, client, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupS3Bucket(context.Background(), client, bucketName)
	})

	putS3Object(t, ctx, client, bucketName, "docs/readme.md", "docs")
	putS3Object(t, ctx, client, bucketName, "docs/specs/v1.txt", "spec")
	putS3Object(t, ctx, client, bucketName, "root.txt", "root")

	backend := NewS3Backend(client, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModalVisible {
		t.Fatal("expected delete modal for directory")
	}
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, S3Location{Mode: S3ModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after directory delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "root.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after directory delete: %v", got)
	}
}

func putS3Object(t *testing.T, ctx context.Context, client *s3.Client, bucket, key, content string) {
	t.Helper()
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(content),
	})
	if err != nil {
		t.Fatalf("put %s/%s: %v", bucket, key, err)
	}
}

func cleanupS3Bucket(ctx context.Context, client *s3.Client, bucket string) {
	pager := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return
		}
		for _, item := range page.Contents {
			if item.Key == nil {
				continue
			}
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(bucket),
				Key:    item.Key,
			})
		}
	}

	_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
}
