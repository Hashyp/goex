package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const s3Delimiter = "/"
const maxS3Entries = 20000
const s3DeleteBatchSize = 1000

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

	return parentHighlightName(s3Location.Prefix, s3Location.Bucket, s3Delimiter)
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

		nextPrefix := enterPrefix(highlighted.FullPath, s3Delimiter)
		return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: nextPrefix}, true, nil
	default:
		return state, false, nil
	}
}

func (b S3Backend) Delete(ctx context.Context, state Location, highlighted Entry) error {
	s3Location, ok := state.(S3Location)
	if !ok {
		return ErrInvalidLocation
	}
	if b.client == nil {
		return fmt.Errorf("s3 client not configured")
	}
	if s3Location.Mode != S3ModeObjects || !isDeleteTargetKind(highlighted.Kind) {
		return nil
	}
	if s3Location.Bucket == "" {
		return fmt.Errorf("s3 bucket not selected")
	}
	key := highlighted.FullPath
	if key == "" {
		key = highlighted.Name
	}
	if key == "" {
		return fmt.Errorf("s3 object key is empty")
	}

	if highlighted.Kind == KindDirectory {
		return b.deletePrefixRecursive(ctx, s3Location.Bucket, key)
	}

	return b.deleteObject(ctx, s3Location.Bucket, key)
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

	parent := parentPrefix(s3Location.Prefix, s3Delimiter)
	return S3Location{Mode: S3ModeObjects, Bucket: s3Location.Bucket, Prefix: parent}, true
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
	return hiddenBySegment(path, s3Delimiter)
}

func strPtr(value string) *string {
	return &value
}

func (b S3Backend) deleteObject(ctx context.Context, bucketName string, key string) error {
	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	return err
}

func (b S3Backend) deletePrefixRecursive(ctx context.Context, bucketName string, prefix string) error {
	queryPrefix := enterPrefix(prefix, s3Delimiter)
	keys, err := b.listKeysByPrefix(ctx, bucketName, queryPrefix)
	if err != nil {
		return err
	}

	// Folder marker object can exist as "dir" in some tools.
	keys = append(keys, prefix, queryPrefix)
	keys = uniqueStrings(keys)
	if len(keys) == 0 {
		return nil
	}

	for start := 0; start < len(keys); start += s3DeleteBatchSize {
		end := start + s3DeleteBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[start:end]

		identifiers := make([]types.ObjectIdentifier, 0, len(batch))
		for _, key := range batch {
			identifiers = append(identifiers, types.ObjectIdentifier{Key: aws.String(key)})
		}

		resp, err := b.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &types.Delete{
				Objects: identifiers,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return err
		}
		if len(resp.Errors) > 0 {
			first := resp.Errors[0]
			return fmt.Errorf("delete object %q failed: %s", aws.ToString(first.Key), aws.ToString(first.Message))
		}
	}

	return nil
}

func (b S3Backend) listKeysByPrefix(ctx context.Context, bucketName string, prefix string) ([]string, error) {
	pager := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	})

	keys := make([]string, 0)
	for pager.HasMorePages() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.Contents {
			if item.Key == nil {
				continue
			}
			keys = append(keys, *item.Key)
		}
	}

	return keys, nil
}

func (b S3Backend) EnumerateCopy(ctx context.Context, state Location, selected []Entry, destination Location) ([]TransferPlanItem, error) {
	s3Location, ok := state.(S3Location)
	if !ok {
		return nil, ErrInvalidLocation
	}
	if b.client == nil {
		return nil, fmt.Errorf("s3 client not configured")
	}

	buildSource := func(scope string, objectPath string) (TransferObjectRef, error) {
		return newTransferObjectRef("s3", scope, objectPath)
	}
	switch s3Location.Mode {
	case S3ModeBuckets:
		return buildObjectStoreCopyPlan(ctx, objectStorePlannerConfig{
			mode:         objectStorePlanModeRoot,
			rootKind:     KindBucket,
			selected:     selected,
			destination:  destination,
			listByPrefix: b.listKeysByPrefix,
			buildSource:  buildSource,
		})
	case S3ModeObjects:
		return buildObjectStoreCopyPlan(ctx, objectStorePlannerConfig{
			mode:         objectStorePlanModeObjects,
			scopeLabel:   "s3 bucket",
			scope:        s3Location.Bucket,
			prefix:       s3Location.Prefix,
			delimiter:    s3Delimiter,
			selected:     selected,
			destination:  destination,
			listByPrefix: b.listKeysByPrefix,
			buildSource:  buildSource,
		})
	default:
		return nil, fmt.Errorf("unknown s3 mode: %s", s3Location.Mode)
	}
}

func (b S3Backend) OpenCopyReader(ctx context.Context, source TransferObjectRef) (CopyReadHandle, error) {
	if b.client == nil {
		return CopyReadHandle{}, fmt.Errorf("s3 client not configured")
	}
	out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(source.Scope),
		Key:    aws.String(source.Path),
	})
	if err != nil {
		return CopyReadHandle{}, err
	}

	handle := CopyReadHandle{Reader: out.Body}
	if out.ContentLength != nil {
		handle.Metadata.SizeBytes = *out.ContentLength
	}
	if out.LastModified != nil {
		handle.Metadata.ModTime = *out.LastModified
		handle.Metadata.HasModTime = true
	}
	return handle, nil
}

func (b S3Backend) CopyDestinationExists(ctx context.Context, destination TransferObjectRef) (bool, error) {
	if b.client == nil {
		return false, fmt.Errorf("s3 client not configured")
	}
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(destination.Scope),
		Key:    aws.String(destination.Path),
	})
	if err == nil {
		return true, nil
	}

	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return false, nil
	}
	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
		return false, nil
	}

	return false, err
}

func (b S3Backend) OpenCopyWriter(ctx context.Context, destination TransferObjectRef, _ TransferObjectMetadata) (io.WriteCloser, error) {
	if b.client == nil {
		return nil, fmt.Errorf("s3 client not configured")
	}

	tmpFile, err := os.CreateTemp("", "goex-copy-s3-*")
	if err != nil {
		return nil, err
	}

	return &s3BufferedUploadWriteCloser{
		ctx:      ctx,
		client:   b.client,
		bucket:   destination.Scope,
		key:      destination.Path,
		file:     tmpFile,
		tempPath: tmpFile.Name(),
	}, nil
}

type pipeUploadWriteCloser struct {
	pipe *io.PipeWriter
	done <-chan error
}

func (w *pipeUploadWriteCloser) Write(p []byte) (int, error) {
	return w.pipe.Write(p)
}

func (w *pipeUploadWriteCloser) Close() error {
	if err := w.pipe.Close(); err != nil {
		return err
	}

	return <-w.done
}

type s3BufferedUploadWriteCloser struct {
	ctx      context.Context
	client   *s3.Client
	bucket   string
	key      string
	file     *os.File
	tempPath string
	closed   bool
}

func (w *s3BufferedUploadWriteCloser) Write(p []byte) (int, error) {
	return w.file.Write(p)
}

func (w *s3BufferedUploadWriteCloser) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	defer func() {
		_ = os.Remove(w.tempPath)
	}()

	info, err := w.file.Stat()
	if err != nil {
		_ = w.file.Close()
		return err
	}
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		_ = w.file.Close()
		return err
	}

	_, uploadErr := w.client.PutObject(w.ctx, &s3.PutObjectInput{
		Bucket:        aws.String(w.bucket),
		Key:           aws.String(w.key),
		Body:          w.file,
		ContentLength: aws.Int64(info.Size()),
	})
	closeErr := w.file.Close()
	if uploadErr != nil {
		return uploadErr
	}
	return closeErr
}
