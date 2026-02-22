package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const gcsDelimiter = "/"
const maxGCSEntries = 20000

type GCSBackend struct {
	client      *storage.Client
	projectID   string
	loadTimeout time.Duration
}

func NewGCSBackend(client *storage.Client, projectID string, loadTimeout time.Duration) GCSBackend {
	if projectID == "" {
		projectID = "goex"
	}
	if loadTimeout <= 0 {
		loadTimeout = 30 * time.Second
	}

	return GCSBackend{client: client, projectID: projectID, loadTimeout: loadTimeout}
}

func (b GCSBackend) InitialLocation() Location {
	return GCSLocation{Mode: GCSModeBuckets}
}

func (b GCSBackend) ParentHighlightName(state Location) string {
	gcsLocation, ok := state.(GCSLocation)
	if !ok || gcsLocation.Mode != GCSModeObjects {
		return ""
	}

	return parentHighlightName(gcsLocation.Prefix, gcsLocation.Bucket, gcsDelimiter)
}

func (b GCSBackend) DisplayPath(state Location) string {
	gcsLocation, ok := state.(GCSLocation)
	if !ok {
		return "gcs:<invalid-location>"
	}

	if gcsLocation.Mode == GCSModeBuckets {
		return "gcs:///"
	}
	if gcsLocation.Prefix == "" {
		return fmt.Sprintf("gcs:///%s", gcsLocation.Bucket)
	}

	return fmt.Sprintf("gcs:///%s/%s", gcsLocation.Bucket, gcsLocation.Prefix)
}

func (b GCSBackend) LoadTimeout() time.Duration {
	return b.loadTimeout
}

func (b GCSBackend) List(ctx context.Context, state Location, showHidden bool) ([]Entry, error) {
	gcsLocation, ok := state.(GCSLocation)
	if !ok {
		return nil, ErrInvalidLocation
	}
	if b.client == nil {
		return nil, fmt.Errorf("gcs client not configured")
	}

	switch gcsLocation.Mode {
	case GCSModeBuckets:
		return b.listBuckets(ctx, showHidden)
	case GCSModeObjects:
		return b.listObjects(ctx, gcsLocation.Bucket, gcsLocation.Prefix, showHidden)
	default:
		return nil, fmt.Errorf("unknown gcs mode: %s", gcsLocation.Mode)
	}
}

func (b GCSBackend) Enter(_ context.Context, state Location, highlighted Entry) (Location, bool, error) {
	gcsLocation, ok := state.(GCSLocation)
	if !ok {
		return state, false, ErrInvalidLocation
	}

	switch gcsLocation.Mode {
	case GCSModeBuckets:
		if highlighted.Kind != KindGCSBucket {
			return state, false, nil
		}
		return GCSLocation{Mode: GCSModeObjects, Bucket: highlighted.Name, Prefix: ""}, true, nil
	case GCSModeObjects:
		if highlighted.Kind != KindDirectory {
			return state, false, nil
		}

		nextPrefix := enterPrefix(highlighted.FullPath, gcsDelimiter)
		return GCSLocation{Mode: GCSModeObjects, Bucket: gcsLocation.Bucket, Prefix: nextPrefix}, true, nil
	default:
		return state, false, nil
	}
}

func (b GCSBackend) Parent(state Location) (Location, bool) {
	gcsLocation, ok := state.(GCSLocation)
	if !ok {
		return state, false
	}

	if gcsLocation.Mode == GCSModeBuckets {
		return state, false
	}
	if gcsLocation.Prefix == "" {
		return GCSLocation{Mode: GCSModeBuckets}, true
	}

	parent := parentPrefix(gcsLocation.Prefix, gcsDelimiter)
	return GCSLocation{Mode: GCSModeObjects, Bucket: gcsLocation.Bucket, Prefix: parent}, true
}

func (b GCSBackend) listBuckets(ctx context.Context, showHidden bool) ([]Entry, error) {
	it := b.client.Buckets(ctx, b.projectID)
	entries := []Entry{}
	for {
		bucketAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		name := bucketAttrs.Name
		if !showHidden && isHiddenByGCSSegment(name) {
			continue
		}

		entry := Entry{
			ID:       "gcs-bucket:" + name,
			Name:     name,
			FullPath: name,
			Kind:     KindGCSBucket,
		}
		if !bucketAttrs.Created.IsZero() {
			entry.ModTime = bucketAttrs.Created
			entry.HasModTime = true
		}

		entries = append(entries, entry)
		if len(entries) > maxGCSEntries {
			return nil, fmt.Errorf("gcs list exceeded max entries limit (%d)", maxGCSEntries)
		}
	}

	sortEntries(entries)
	return entries, nil
}

func (b GCSBackend) listObjects(ctx context.Context, bucketName string, prefix string, showHidden bool) ([]Entry, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("gcs bucket not selected")
	}

	it := b.client.Bucket(bucketName).Objects(ctx, &storage.Query{Prefix: prefix, Delimiter: gcsDelimiter})
	entries := []Entry{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if attrs.Prefix != "" {
			fullPrefix := strings.TrimSuffix(attrs.Prefix, gcsDelimiter)
			displayName := trimPrefix(fullPrefix, prefix)
			displayName = strings.TrimSuffix(displayName, gcsDelimiter)
			if displayName == "" {
				continue
			}
			if !showHidden && isHiddenByGCSSegment(displayName) {
				continue
			}

			entries = append(entries, Entry{
				ID:       "gcs-dir:" + bucketName + "/" + fullPrefix,
				Name:     displayName,
				FullPath: fullPrefix,
				Kind:     KindDirectory,
			})
			if len(entries) > maxGCSEntries {
				return nil, fmt.Errorf("gcs list exceeded max entries limit (%d)", maxGCSEntries)
			}
			continue
		}

		if attrs.Name == "" {
			continue
		}

		fullName := attrs.Name
		displayName := trimPrefix(fullName, prefix)
		if displayName == "" || strings.Contains(displayName, gcsDelimiter) {
			continue
		}
		if !showHidden && isHiddenByGCSSegment(displayName) {
			continue
		}

		entry := Entry{
			ID:        "gcs-object:" + bucketName + "/" + fullName,
			Name:      displayName,
			FullPath:  fullName,
			Kind:      KindObject,
			SizeBytes: attrs.Size,
		}
		if !attrs.Updated.IsZero() {
			entry.ModTime = attrs.Updated
			entry.HasModTime = true
		}

		entries = append(entries, entry)
		if len(entries) > maxGCSEntries {
			return nil, fmt.Errorf("gcs list exceeded max entries limit (%d)", maxGCSEntries)
		}
	}

	sortEntries(entries)
	return entries, nil
}

func isHiddenByGCSSegment(path string) bool {
	return hiddenBySegment(path, gcsDelimiter)
}
