package app

import (
	"context"
	"testing"
)

func TestGCSEnterFromBucketListToBucketRoot(t *testing.T) {
	backend := NewGCSBackend(nil, "goex", 0)
	start := GCSLocation{Mode: GCSModeBuckets}

	next, changed, err := backend.Enter(context.Background(), start, Entry{Name: "goex-dev", Kind: KindGCSBucket})
	if err != nil {
		t.Fatalf("enter returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected bucket enter to change location")
	}

	gcsloc, ok := next.(GCSLocation)
	if !ok {
		t.Fatalf("unexpected location type: %T", next)
	}
	if gcsloc.Mode != GCSModeObjects || gcsloc.Bucket != "goex-dev" || gcsloc.Prefix != "" {
		t.Fatalf("unexpected next location: %+v", gcsloc)
	}
}

func TestGCSParentTransitions(t *testing.T) {
	backend := NewGCSBackend(nil, "goex", 0)

	loc := GCSLocation{Mode: GCSModeObjects, Bucket: "goex-dev", Prefix: "docs/specs/"}
	parent, changed := backend.Parent(loc)
	if !changed {
		t.Fatal("expected parent change for nested prefix")
	}
	gcsloc, _ := parent.(GCSLocation)
	if gcsloc.Prefix != "docs/" {
		t.Fatalf("unexpected first parent prefix: %q", gcsloc.Prefix)
	}

	parent, changed = backend.Parent(gcsloc)
	if !changed {
		t.Fatal("expected second parent change")
	}
	gcsloc, _ = parent.(GCSLocation)
	if gcsloc.Prefix != "" || gcsloc.Mode != GCSModeObjects {
		t.Fatalf("unexpected bucket root location: %+v", gcsloc)
	}

	parent, changed = backend.Parent(gcsloc)
	if !changed {
		t.Fatal("expected transition from bucket root to buckets mode")
	}
	gcsloc, _ = parent.(GCSLocation)
	if gcsloc.Mode != GCSModeBuckets {
		t.Fatalf("expected buckets mode, got %+v", gcsloc)
	}
}

func TestGCSHiddenSegmentDetection(t *testing.T) {
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
		if got := isHiddenByGCSSegment(tc.path); got != tc.hidden {
			t.Fatalf("hidden mismatch for %q: got %v want %v", tc.path, got, tc.hidden)
		}
	}
}

func TestGCSDisplayPathConvention(t *testing.T) {
	backend := NewGCSBackend(nil, "goex", 0)

	if got := backend.DisplayPath(GCSLocation{Mode: GCSModeBuckets}); got != "gcs:///" {
		t.Fatalf("unexpected root path: %q", got)
	}
	if got := backend.DisplayPath(GCSLocation{Mode: GCSModeObjects, Bucket: "bucket"}); got != "gcs:///bucket" {
		t.Fatalf("unexpected bucket path: %q", got)
	}
	if got := backend.DisplayPath(GCSLocation{Mode: GCSModeObjects, Bucket: "bucket", Prefix: "docs/"}); got != "gcs:///bucket/docs/" {
		t.Fatalf("unexpected prefix path: %q", got)
	}
}

func TestGCSListRequiresClient(t *testing.T) {
	backend := NewGCSBackend(nil, "goex", 0)
	_, err := backend.List(context.Background(), GCSLocation{Mode: GCSModeBuckets}, true)
	if err == nil {
		t.Fatal("expected error when gcs client is nil")
	}
}
