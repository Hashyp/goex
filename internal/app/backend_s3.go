package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const s3Delimiter = "/"
const maxS3Entries = 20000

type S3Backend struct {
	client      *s3.Client
	loadTimeout time.Duration
}

func NewS3Backend(client *s3.Client, loadTimeout time.Duration) S3Backend {
	if loadTimeout <= 0 {
		loadTimeout = 30 * time.Second
	}

	return S3Backend{
		client:      client,
		loadTimeout: loadTimeout,
	}
}

func (b S3Backend) InitialLocation() Location {
	return S3Location{Mode: S3ModeBuckets}
}

func (b S3Backend) ParentHighlightName(state Location) string {
	s3Location, ok := state.(S3Location)
	if !ok || s3Location.Mode != S3ModeObjects {
		return ""
	}

	trimmed := strings.TrimSuffix(s3Location.Prefix, s3Delimiter)
	if trimmed == "" {
		return s3Location.Bucket
	}

	parts := strings.Split(trimmed, s3Delimiter)
	return parts[len(parts)-1]
}

func (b S3Backend) DisplayPath(state Location) string {
	s3Location, ok := state.(S3Location)
	if !ok {
		return "s3:<invalid-location>"
	}

	if s3Location.Mode == S3ModeBuckets {
		return "s3:///"
	}
	if s3Location.Prefix == "" {
		return fmt.Sprintf("s3:///%s", s3Location.Bucket)
	}

	return fmt.Sprintf("s3:///%s/%s", s3Location.Bucket, s3Location.Prefix)
}

func (b S3Backend) LoadTimeout() time.Duration {
	return b.loadTimeout
}

func (b S3Backend) List(ctx context.Context, state Location, showHidden bool) ([]Entry, error) {
	s3Location, ok := state.(S3Location)
	if !ok {
		return nil, ErrInvalidLocation
	}
	if b.client == nil {
		return nil, fmt.Errorf("s3 client not configured")
	}

	switch s3Location.Mode {
	case S3ModeBuckets:
		return b.listBuckets(ctx, showHidden)
	case S3ModeObjects:
		return b.listObjects(ctx, s3Location.Bucket, s3Location.Prefix, showHidden)
	default:
		return nil, fmt.Errorf("unknown s3 mode: %s", s3Location.Mode)
	}
}

func (b S3Backend) Enter(_ context.Context, state Location, highlighted Entry) (Location, bool, error) {
	s3Location, ok := state.(S3Location)
	if !ok {
		return state, false, ErrInvalidLocation
	}

	switch s3Location.Mode {
	case S3ModeBuckets:
		if highlighted.Kind != KindBucket {
			return state, false, nil
		}
		return S3Location{Mode: S3ModeObjects, Bucket: highlighted.Name, Prefix: ""}, true, nil
	case S3ModeObjects:
		if highlighted.Kind != KindDirectory {
			return state, false, nil
		}

		nextPrefix := highlighted.FullPath
		if nextPrefix != "" && !strings.HasSuffix(nextPrefix, s3Delimiter) {
			nextPrefix += s3Delimiter
		}
		return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: nextPrefix}, true, nil
	default:
		return state, false, nil
	}
}

func (b S3Backend) Parent(state Location) (Location, bool) {
	s3Location, ok := state.(S3Location)
	if !ok {
		return state, false
	}

	if s3Location.Mode == S3ModeBuckets {
		return state, false
	}
	if s3Location.Prefix == "" {
		return S3Location{Mode: S3ModeBuckets}, true
	}

	trimmed := strings.TrimSuffix(s3Location.Prefix, s3Delimiter)
	if trimmed == "" {
		return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: ""}, true
	}

	lastSlash := strings.LastIndex(trimmed, s3Delimiter)
	if lastSlash < 0 {
		return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: ""}, true
	}

	parentPrefix := trimmed[:lastSlash+1]
	return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: parentPrefix}, true
}

func (b S3Backend) listBuckets(ctx context.Context, showHidden bool) ([]Entry, error) {
	out, err := b.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(out.Buckets))
	for _, bucket := range out.Buckets {
		if bucket.Name == nil {
			continue
		}

		name := *bucket.Name
		if !showHidden && isHiddenByS3Segment(name) {
			continue
		}

		entry := Entry{
			ID:       "bucket:" + name,
			Name:     name,
			FullPath: name,
			Kind:     KindBucket,
		}
		if bucket.CreationDate != nil {
			entry.ModTime = *bucket.CreationDate
			entry.HasModTime = true
		}

		entries = append(entries, entry)
		if len(entries) > maxS3Entries {
			return nil, fmt.Errorf("s3 list exceeded max entries limit (%d)", maxS3Entries)
		}
	}

	sortEntries(entries)
	return entries, nil
}

func (b S3Backend) listObjects(ctx context.Context, bucketName string, prefix string, showHidden bool) ([]Entry, error) {
	if bucketName == "" {
		return nil, fmt.Errorf("s3 bucket not selected")
	}

	pager := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket:    &bucketName,
		Delimiter: strPtr(s3Delimiter),
		Prefix:    strPtr(prefix),
	})

	entries := []Entry{}
	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range resp.CommonPrefixes {
			if item.Prefix == nil {
				continue
			}

			fullPrefix := strings.TrimSuffix(*item.Prefix, s3Delimiter)
			displayName := trimPrefix(fullPrefix, prefix)
			displayName = strings.TrimSuffix(displayName, s3Delimiter)
			if displayName == "" {
				continue
			}
			if !showHidden && isHiddenByS3Segment(displayName) {
				continue
			}

			entries = append(entries, Entry{
				ID:       "dir:" + bucketName + "/" + fullPrefix,
				Name:     displayName,
				FullPath: fullPrefix,
				Kind:     KindDirectory,
			})
			if len(entries) > maxS3Entries {
				return nil, fmt.Errorf("s3 list exceeded max entries limit (%d)", maxS3Entries)
			}
		}

		for _, item := range resp.Contents {
			if item.Key == nil {
				continue
			}

			fullName := *item.Key
			displayName := trimPrefix(fullName, prefix)
			if displayName == "" || strings.Contains(displayName, s3Delimiter) {
				continue
			}
			if !showHidden && isHiddenByS3Segment(displayName) {
				continue
			}

			entry := Entry{
				ID:       "object:" + bucketName + "/" + fullName,
				Name:     displayName,
				FullPath: fullName,
				Kind:     KindObject,
			}
			if item.Size != nil {
				entry.SizeBytes = *item.Size
			}
			if item.LastModified != nil {
				entry.ModTime = *item.LastModified
				entry.HasModTime = true
			}

			entries = append(entries, entry)
			if len(entries) > maxS3Entries {
				return nil, fmt.Errorf("s3 list exceeded max entries limit (%d)", maxS3Entries)
			}
		}
	}

	sortEntries(entries)
	return entries, nil
}

func isHiddenByS3Segment(path string) bool {
	for _, segment := range strings.Split(path, s3Delimiter) {
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}

	return false
}

func strPtr(value string) *string {
	return &value
}
