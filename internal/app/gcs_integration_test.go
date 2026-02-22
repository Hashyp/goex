package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/iterator"

	"defaultdevcontainer/internal/gcsblob"
)

func TestGCSBackendListsBucketsAndVirtualFolders(t *testing.T) {
	if os.Getenv("GOEX_RUN_GCS_TESTS") != "1" {
		t.Skip("set GOEX_RUN_GCS_TESTS=1 to run GCS emulator integration tests")
	}

	ctx := context.Background()
	cfg := gcsblob.DefaultConfig()
	client, err := gcsblob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create gcs client: %v", err)
	}
	defer client.Close()

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := gcsblob.EnsureBucket(ctx, client, cfg.ProjectID, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupGCSBucket(context.Background(), client, bucketName)
	})

	putGCSObject(t, ctx, client, bucketName, "root.txt", "root")
	putGCSObject(t, ctx, client, bucketName, "docs/readme.md", "docs")
	putGCSObject(t, ctx, client, bucketName, "docs/specs/v1.txt", "spec")
	putGCSObject(t, ctx, client, bucketName, "configs/.secrets/app.env", "hidden-segment")
	putGCSObject(t, ctx, client, bucketName, ".hidden-root.txt", "hidden-root")

	backend := NewGCSBackend(client, cfg.ProjectID, cfg.RequestTimeout)

	buckets, err := backend.List(ctx, GCSLocation{Mode: GCSModeBuckets}, true)
	if err != nil {
		t.Fatalf("list buckets: %v", err)
	}

	var foundBucket bool
	for _, entry := range buckets {
		if entry.Name == bucketName && entry.Kind == KindGCSBucket {
			foundBucket = true
			break
		}
	}
	if !foundBucket {
		t.Fatalf("expected to find seeded bucket %q", bucketName)
	}

	rootLocation := GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}
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

	docsLocation := GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: "docs/"}
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

func TestGCSModelDeleteSelectedFiles(t *testing.T) {
	if os.Getenv("GOEX_RUN_GCS_TESTS") != "1" {
		t.Skip("set GOEX_RUN_GCS_TESTS=1 to run GCS emulator integration tests")
	}

	ctx := context.Background()
	cfg := gcsblob.DefaultConfig()
	client, err := gcsblob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create gcs client: %v", err)
	}
	defer client.Close()

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := gcsblob.EnsureBucket(ctx, client, cfg.ProjectID, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupGCSBucket(context.Background(), client, bucketName)
	})

	putGCSObject(t, ctx, client, bucketName, "a.txt", "a")
	putGCSObject(t, ctx, client, bucketName, "b.txt", "b")
	putGCSObject(t, ctx, client, bucketName, "c.txt", "c")

	backend := NewGCSBackend(client, cfg.ProjectID, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "c.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after selected delete: %v", got)
	}
}

func TestGCSModelDeleteSingleHighlightedFile(t *testing.T) {
	if os.Getenv("GOEX_RUN_GCS_TESTS") != "1" {
		t.Skip("set GOEX_RUN_GCS_TESTS=1 to run GCS emulator integration tests")
	}

	ctx := context.Background()
	cfg := gcsblob.DefaultConfig()
	client, err := gcsblob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create gcs client: %v", err)
	}
	defer client.Close()

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := gcsblob.EnsureBucket(ctx, client, cfg.ProjectID, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupGCSBucket(context.Background(), client, bucketName)
	})

	putGCSObject(t, ctx, client, bucketName, "a.txt", "a")
	putGCSObject(t, ctx, client, bucketName, "b.txt", "b")

	backend := NewGCSBackend(client, cfg.ProjectID, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "b.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after single delete: %v", got)
	}
}

func TestGCSModelDeleteDirectoryRecursively(t *testing.T) {
	if os.Getenv("GOEX_RUN_GCS_TESTS") != "1" {
		t.Skip("set GOEX_RUN_GCS_TESTS=1 to run GCS emulator integration tests")
	}

	ctx := context.Background()
	cfg := gcsblob.DefaultConfig()
	client, err := gcsblob.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("create gcs client: %v", err)
	}
	defer client.Close()

	bucketName := fmt.Sprintf("goex-it-%d", time.Now().UnixNano())
	if err := gcsblob.EnsureBucket(ctx, client, cfg.ProjectID, bucketName); err != nil {
		t.Fatalf("ensure test bucket: %v", err)
	}
	t.Cleanup(func() {
		cleanupGCSBucket(context.Background(), client, bucketName)
	})

	putGCSObject(t, ctx, client, bucketName, "docs/readme.md", "docs")
	putGCSObject(t, ctx, client, bucketName, "docs/specs/v1.txt", "spec")
	putGCSObject(t, ctx, client, bucketName, "root.txt", "root")

	backend := NewGCSBackend(client, cfg.ProjectID, cfg.RequestTimeout)
	model := NewModelWithBackends(backend, backend)
	model.leftPane.location = GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}
	model.leftPane.path = backend.DisplayPath(model.leftPane.location)
	model = runCmd(t, model, model.leftPane.beginLoad(paneLeft))

	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !model.deleteModal.visible {
		t.Fatal("expected delete modal for directory")
	}
	model = pressKey(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	remaining, err := backend.List(ctx, GCSLocation{Mode: GCSModeObjects, Bucket: bucketName, Prefix: ""}, true)
	if err != nil {
		t.Fatalf("list after directory delete: %v", err)
	}
	if got := entryNames(remaining); !contains(got, "root.txt") || len(got) != 1 {
		t.Fatalf("unexpected entries after directory delete: %v", got)
	}
}

func putGCSObject(t *testing.T, ctx context.Context, client *storage.Client, bucket, key, content string) {
	t.Helper()
	writer := client.Bucket(bucket).Object(key).NewWriter(ctx)
	if _, err := writer.Write([]byte(content)); err != nil {
		_ = writer.Close()
		t.Fatalf("write %s/%s: %v", bucket, key, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer %s/%s: %v", bucket, key, err)
	}
}

func cleanupGCSBucket(ctx context.Context, client *storage.Client, bucket string) {
	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		if attrs.Name == "" {
			continue
		}
		_ = client.Bucket(bucket).Object(attrs.Name).Delete(ctx)
	}

	_ = client.Bucket(bucket).Delete(ctx)
}
