package app

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"defaultdevcontainer/internal/gcsblob"
	"defaultdevcontainer/internal/s3blob"
)

func TestCopyGCSDirectoryToS3Prefix(t *testing.T) {
	if os.Getenv("GOEX_RUN_GCS_TESTS") != "1" || os.Getenv("GOEX_RUN_MINIO_TESTS") != "1" {
		t.Skip("set GOEX_RUN_GCS_TESTS=1 and GOEX_RUN_MINIO_TESTS=1 to run cross-backend copy integration tests")
	}

	ctx := context.Background()

	gcsCfg := gcsblob.DefaultConfig()
	gcsClient, err := gcsblob.NewClient(ctx, gcsCfg)
	if err != nil {
		t.Fatalf("create gcs client: %v", err)
	}
	defer gcsClient.Close()

	s3Cfg := s3blob.DefaultConfig()
	s3Client, err := s3blob.NewClient(ctx, s3Cfg)
	if err != nil {
		t.Fatalf("create s3 client: %v", err)
	}

	srcBucket := fmt.Sprintf("goex-copy-src-%d", time.Now().UnixNano())
	if err := gcsblob.EnsureBucket(ctx, gcsClient, gcsCfg.ProjectID, srcBucket); err != nil {
		t.Fatalf("ensure gcs source bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupGCSBucket(context.Background(), gcsClient, srcBucket)
	})

	dstBucket := fmt.Sprintf("goex-copy-dst-%d", time.Now().UnixNano())
	if err := s3blob.EnsureBucket(ctx, s3Client, dstBucket); err != nil {
		t.Fatalf("ensure s3 destination bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupS3Bucket(context.Background(), s3Client, dstBucket)
	})

	putGCSObject(t, ctx, gcsClient, srcBucket, "docs/readme.md", "readme")
	putGCSObject(t, ctx, gcsClient, srcBucket, "docs/specs/v1.txt", "v1")
	if reader, err := gcsClient.Bucket(srcBucket).Object("docs/readme.md").NewReader(ctx); err != nil {
		t.Fatalf("sanity read source object: %v", err)
	} else {
		_ = reader.Close()
	}

	sourceBackend := NewGCSBackend(gcsClient, gcsCfg.ProjectID, gcsCfg.RequestTimeout)
	targetBackend := NewS3Backend(s3Client, s3Cfg.RequestTimeout)

	sourceLocation := GCSLocation{Mode: GCSModeObjects, Bucket: srcBucket, Prefix: ""}
	entries, err := sourceBackend.List(ctx, sourceLocation, true)
	if err != nil {
		t.Fatalf("list gcs source: %v", err)
	}

	var selected Entry
	for _, entry := range entries {
		if entry.Name == "docs" && entry.Kind == KindDirectory {
			selected = entry
			break
		}
	}
	if selected.Name == "" {
		t.Fatalf("expected docs directory in source entries: %v", entryNames(entries))
	}

	destination := S3Location{Mode: S3ModeObjects, Bucket: dstBucket, Prefix: "target/"}
	plan, err := sourceBackend.EnumerateCopy(ctx, sourceLocation, []Entry{selected}, destination)
	if err != nil {
		t.Fatalf("enumerate copy plan: %v", err)
	}
	if len(plan) == 0 {
		t.Fatal("expected non-empty copy plan")
	}

	result := ExecuteCopy(ctx, TransferCopyRequest{
		Plan:           plan,
		ConflictPolicy: TransferConflictOverwrite,
	}, sourceBackend, targetBackend)
	if len(result.Failed) > 0 {
		first := result.Failed[0]
		t.Fatalf("copy failed: stage=%s src=%+v dst=%+v err=%v", first.Stage, first.PlanItem.Source, first.PlanItem.Destination, first.Err)
	}
}
