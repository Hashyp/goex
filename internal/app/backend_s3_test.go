package app

import (
	"context"
	"testing"
)

func TestS3EnterFromBucketListToBucketRoot(t *testing.T) {
	backend := NewS3Backend(nil, 0)
	start := S3Location{Mode: S3ModeBuckets}

	next, changed, err := backend.Enter(context.Background(), start, Entry{Name: "goex-dev", Kind: KindBucket})
	if err != nil {
		t.Fatalf("enter returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected bucket enter to change location")
	}

	s3loc, ok := next.(S3Location)
	if !ok {
		t.Fatalf("unexpected location type: %T", next)
	}
	if s3loc.Mode != S3ModeObjects || s3loc.Bucket != "goex-dev" || s3loc.Prefix != "" {
		t.Fatalf("unexpected next location: %+v", s3loc)
	}
}

func TestS3ParentTransitions(t *testing.T) {
	backend := NewS3Backend(nil, 0)

	loc := S3Location{Mode: S3ModeObjects, Bucket: "goex-dev", Prefix: "docs/specs/"}
	parent, changed := backend.Parent(loc)
	if !changed {
		t.Fatal("expected parent change for nested prefix")
	}
	s3loc, _ := parent.(S3Location)
	if s3loc.Prefix != "docs/" {
		t.Fatalf("unexpected first parent prefix: %q", s3loc.Prefix)
	}

	parent, changed = backend.Parent(s3loc)
	if !changed {
		t.Fatal("expected second parent change")
	}
	s3loc, _ = parent.(S3Location)
	if s3loc.Prefix != "" || s3loc.Mode != S3ModeObjects {
		t.Fatalf("unexpected bucket root location: %+v", s3loc)
	}

	parent, changed = backend.Parent(s3loc)
	if !changed {
		t.Fatal("expected transition from bucket root to buckets mode")
	}
	s3loc, _ = parent.(S3Location)
	if s3loc.Mode != S3ModeBuckets {
		t.Fatalf("expected buckets mode, got %+v", s3loc)
	}
}

func TestS3HiddenSegmentDetection(t *testing.T) {
	cases := []struct {
		path   string
		hidden bool
	}{
		{path: "plain.txt", hidden: false},
		{path: ".env", hidden: true},
		{path: "docs/.secret/file.txt", hidden: true},
		{path: "docs/specs/file.txt", hidden: false},
	}

	for _, tc := range cases {
		if got := isHiddenByS3Segment(tc.path); got != tc.hidden {
			t.Fatalf("hidden mismatch for %q: got %v want %v", tc.path, got, tc.hidden)
		}
	}
}

func TestS3DisplayPathConvention(t *testing.T) {
	backend := NewS3Backend(nil, 0)

	if got := backend.DisplayPath(S3Location{Mode: S3ModeBuckets}); got != "s3:///" {
		t.Fatalf("unexpected root path: %q", got)
	}
	if got := backend.DisplayPath(S3Location{Mode: S3ModeObjects, Bucket: "bucket"}); got != "s3:///bucket" {
		t.Fatalf("unexpected bucket path: %q", got)
	}
	if got := backend.DisplayPath(S3Location{Mode: S3ModeObjects, Bucket: "bucket", Prefix: "docs/"}); got != "s3:///bucket/docs/" {
		t.Fatalf("unexpected prefix path: %q", got)
	}
}

func TestS3ListRequiresClient(t *testing.T) {
	backend := NewS3Backend(nil, 0)
	_, err := backend.List(context.Background(), S3Location{Mode: S3ModeBuckets}, true)
	if err == nil {
		t.Fatal("expected error when s3 client is nil")
	}
}
