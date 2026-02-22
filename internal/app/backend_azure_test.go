package app

import (
	"context"
	"errors"
	"testing"
)

func TestAzureEnterFromContainerListToContainerRoot(t *testing.T) {
	backend := NewAzureBlobBackend(nil)
	start := AzureLocation{Mode: AzureModeContainers}

	next, changed, err := backend.Enter(context.Background(), start, Entry{Name: "goex-dev", Kind: KindContainer})
	if err != nil {
		t.Fatalf("enter returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected container enter to change location")
	}

	azure, ok := next.(AzureLocation)
	if !ok {
		t.Fatalf("unexpected location type: %T", next)
	}
	if azure.Mode != AzureModeObjects || azure.Container != "goex-dev" || azure.Prefix != "" {
		t.Fatalf("unexpected next location: %+v", azure)
	}
}

func TestAzureParentTransitions(t *testing.T) {
	backend := NewAzureBlobBackend(nil)

	loc := AzureLocation{Mode: AzureModeObjects, Container: "goex-dev", Prefix: "docs/specs/"}
	parent, changed := backend.Parent(loc)
	if !changed {
		t.Fatal("expected parent change for nested prefix")
	}
	azure, _ := parent.(AzureLocation)
	if azure.Prefix != "docs/" {
		t.Fatalf("unexpected first parent prefix: %q", azure.Prefix)
	}

	parent, changed = backend.Parent(azure)
	if !changed {
		t.Fatal("expected second parent change")
	}
	azure, _ = parent.(AzureLocation)
	if azure.Prefix != "" || azure.Mode != AzureModeObjects {
		t.Fatalf("unexpected container root location: %+v", azure)
	}

	parent, changed = backend.Parent(azure)
	if !changed {
		t.Fatal("expected transition from container root to containers mode")
	}
	azure, _ = parent.(AzureLocation)
	if azure.Mode != AzureModeContainers {
		t.Fatalf("expected containers mode, got %+v", azure)
	}
}

func TestAzureHiddenSegmentDetection(t *testing.T) {
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
		if got := isHiddenBySegment(tc.path); got != tc.hidden {
			t.Fatalf("hidden mismatch for %q: got %v want %v", tc.path, got, tc.hidden)
		}
	}
}

func TestAzureListRequiresClient(t *testing.T) {
	backend := NewAzureBlobBackend(nil)
	_, err := backend.List(context.Background(), AzureLocation{Mode: AzureModeContainers}, true)
	if err == nil {
		t.Fatal("expected error when azure client is nil")
	}
}

func TestStaticErrorBackendReturnsConfiguredError(t *testing.T) {
	expected := errors.New("boom")
	backend := NewStaticErrorBackend(expected)
	_, err := backend.List(context.Background(), backend.InitialLocation(), true)
	if !errors.Is(err, expected) {
		t.Fatalf("expected configured error, got %v", err)
	}
}
